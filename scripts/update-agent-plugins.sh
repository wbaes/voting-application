#!/usr/bin/env bash
#
# update-agent-plugins.sh — Fetch and install shared agent plugins and commands
#
# Reads agent-plugins.conf from the repository root, resolves versions from
# a GitLab instance, downloads skills and commands, and installs them for
# GitHub Copilot and Cursor.
#
# Usage:
#   scripts/update-agent-plugins.sh [OPTIONS]
#
# Options:
#   --background   Redirect output to a log file (used by git hooks)
#   --force        Force re-download and reinstall even if cached
#   --verbose      Show detailed progress output
#   --help         Show this help message
#
# Exit codes:
#   0  Success (updated or already up to date)
#   1  Error (missing config, network failure without cache, etc.)

set -euo pipefail

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------

readonly CONFIG_FILE="agent-plugins.conf"
readonly STATE_DIR=".agent-plugins"
readonly MANIFEST_FILE="${STATE_DIR}/manifest"
readonly INSTALLED_FILE="${STATE_DIR}/installed-versions"
readonly LOCK_DIR="${STATE_DIR}/.lock"
readonly LOG_FILE="${STATE_DIR}/update.log"
readonly SHARED_PREFIX="_shared"
readonly INSTALL_FORMAT_VERSION="2"

# ---------------------------------------------------------------------------
# Globals
# ---------------------------------------------------------------------------

BACKGROUND=false
FORCE=false
VERBOSE=false
GITLAB_URL=""
PROJECT_PATH=""
PROJECT_ID=""
SKILLS=()
SKILL_VERSIONS=()
COMMANDS=()
COMMAND_VERSIONS=()



# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------

while [[ $# -gt 0 ]]; do
    case "$1" in
        --background) BACKGROUND=true; shift ;;
        --force)      FORCE=true; shift ;;
        --verbose)    VERBOSE=true; shift ;;
        --help)
            sed -n '2,/^[^#]/p' "$0" | grep '^#' | sed 's/^# \?//'
            exit 0
            ;;
        *) echo "Unknown option: $1" >&2; exit 1 ;;
    esac
done

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------

log() {
    local msg
    msg="[agent-plugins] $(date '+%H:%M:%S') $*"
    if [[ "$BACKGROUND" == true ]]; then
        echo "$msg" >> "$LOG_FILE"
    else
        echo "$msg"
    fi
}

log_verbose() {
    if [[ "$VERBOSE" == true ]]; then
        log "$@"
    fi
    return 0
}

die() {
    log "ERROR: $*"
    exit 1
}

# ---------------------------------------------------------------------------
# Config parsing
# ---------------------------------------------------------------------------

parse_config() {
    [[ -f "$CONFIG_FILE" ]] || die "Config file not found: $CONFIG_FILE"

    while IFS= read -r line || [[ -n "$line" ]]; do
        # Strip inline comments and trim whitespace
        line="${line%%#*}"
        line="$(printf '%s' "$line" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"
        [[ -z "$line" ]] && continue

        case "$line" in
            GITLAB_URL=*)   GITLAB_URL="${line#GITLAB_URL=}" ;;
            PROJECT_PATH=*) PROJECT_PATH="${line#PROJECT_PATH=}" ;;
            command:*)      COMMANDS+=("${line#command:}") ;;
            *=*)            log_verbose "Ignoring unknown config key: ${line%%=*}" ;;
            *)              SKILLS+=("$line") ;;
        esac
    done < "$CONFIG_FILE"

    # Validate required fields
    [[ -n "$GITLAB_URL" ]]   || die "GITLAB_URL not set in $CONFIG_FILE"
    [[ -n "$PROJECT_PATH" ]] || die "PROJECT_PATH not set in $CONFIG_FILE"
    if [[ ${#SKILLS[@]} -eq 0 ]] && [[ ${#COMMANDS[@]} -eq 0 ]]; then
        die "No skills or commands listed in $CONFIG_FILE"
    fi

    # Remove trailing slash from URL
    GITLAB_URL="${GITLAB_URL%/}"

    # URL-encode the project path for API calls (replace / with %2F)
    PROJECT_ID="$(printf '%s' "$PROJECT_PATH" | sed 's/\//%2F/g')"
}

# ---------------------------------------------------------------------------
# GitLab API
# ---------------------------------------------------------------------------

gitlab_api() {
    local endpoint="$1"
    curl -sf --max-time 15 \
        "${GITLAB_URL}/api/v4/projects/${PROJECT_ID}${endpoint}" 2>/dev/null
}

# Resolve the latest release tag, with multiple fallback strategies.
resolve_latest_tag() {
    local tag=""

    # Strategy 1: permalink/latest (GitLab 15.7+)
    tag="$(gitlab_api "/releases/permalink/latest" \
        | grep -o '"tag_name":"[^"]*"' | head -1 | cut -d'"' -f4)" || true
    if [[ -n "$tag" ]]; then echo "$tag"; return 0; fi

    # Strategy 2: list releases, take first (sorted by released_at desc)
    tag="$(gitlab_api "/releases?per_page=1" \
        | grep -o '"tag_name":"[^"]*"' | head -1 | cut -d'"' -f4)" || true
    if [[ -n "$tag" ]]; then echo "$tag"; return 0; fi

    # Strategy 3: list tags sorted by version
    tag="$(gitlab_api "/repository/tags?per_page=1&order_by=version" \
        | grep -o '"name":"[^"]*"' | head -1 | cut -d'"' -f4)" || true
    if [[ -n "$tag" ]]; then echo "$tag"; return 0; fi

    return 1
}

# ---------------------------------------------------------------------------
# Archive management
# ---------------------------------------------------------------------------

download_archive() {
    local tag="$1"
    local archive_file="${STATE_DIR}/archive/${tag}.tar.gz"

    mkdir -p "${STATE_DIR}/archive"

    if [[ -f "$archive_file" ]] && [[ "$FORCE" != true ]]; then
        log_verbose "Using cached archive for $tag"
        return 0
    fi

    log "Downloading archive for ${tag}..."
    local tmp_file="${archive_file}.tmp.$$"

    if ! curl -sfL --max-time 120 \
        "${GITLAB_URL}/api/v4/projects/${PROJECT_ID}/repository/archive.tar.gz?sha=${tag}" \
        -o "$tmp_file"; then
        rm -f "$tmp_file"
        return 1
    fi

    mv "$tmp_file" "$archive_file"
    return 0
}

extract_archive() {
    local tag="$1"
    local archive_file="${STATE_DIR}/archive/${tag}.tar.gz"
    local extract_dir="${STATE_DIR}/extracted/${tag}"

    if [[ -d "$extract_dir/skills" ]] || [[ -d "$extract_dir/commands" ]]; then
        if [[ "$FORCE" != true ]]; then
            log_verbose "Using cached extraction for $tag"
            return 0
        fi
    fi

    rm -rf "$extract_dir"
    mkdir -p "$extract_dir"

    # GitLab archives have a top-level directory (project-tag-hash/).
    # --strip-components=1 removes it so we get skills/, client/, etc. directly.
    tar -xzf "$archive_file" -C "$extract_dir" --strip-components=1

    if [[ ! -d "$extract_dir/skills" ]] && [[ ! -d "$extract_dir/commands" ]]; then
        die "Archive for $tag contains neither skills/ nor commands/"
    fi
}

# ---------------------------------------------------------------------------
# Git hooks
# ---------------------------------------------------------------------------

install_hooks() {
    local hooks_dir
    hooks_dir="$(git config core.hooksPath 2>/dev/null)" || true
    if [[ -z "$hooks_dir" ]]; then
        hooks_dir=".git/hooks"
    fi

    mkdir -p "$hooks_dir"

    local marker="# Shared agent plugins"

    local hook_code
    hook_code='# Shared agent plugins — auto-update after checkout/merge
# Runs in background to avoid blocking git operations.
if [[ -f "agent-plugins.conf" ]] && [[ -x "scripts/update-agent-plugins.sh" ]]; then
    bash scripts/update-agent-plugins.sh --background &
    disown 2>/dev/null || true
fi'

    install_single_hook "$hooks_dir/post-checkout" "$hook_code" "$marker"
    install_single_hook "$hooks_dir/post-merge" "$hook_code" "$marker"
}

install_single_hook() {
    local hook_file="$1"
    local hook_code="$2"
    local marker="$3"
    local hook_name
    hook_name="$(basename "$hook_file")"

    if [[ -f "$hook_file" ]]; then
        if grep -qF "$marker" "$hook_file" 2>/dev/null; then
            log_verbose "Hook ${hook_name}: already configured"
            return 0
        fi
        {
            echo ""
            echo "$hook_code"
        } >>"$hook_file"
        chmod +x "$hook_file"
        log_verbose "Hook ${hook_name}: appended skill-update trigger"
    else
        {
            echo "#!/usr/bin/env bash"
            echo ""
            echo "$hook_code"
        } >"$hook_file"
        chmod +x "$hook_file"
        log_verbose "Hook ${hook_name}: created"
    fi
}

# ---------------------------------------------------------------------------
# Agent installers
# ---------------------------------------------------------------------------

# Install a skill using the Agent Skills spec.
# Copies SKILL.md and any additional files (scripts/, references/, assets/, etc.)
# from the source directory into .agents/skills/<name>/.
install_skill() {
    local skill_name="$1"
    local skill_dir="$2"

    local content_file="${skill_dir}/SKILL.md"
    [[ -f "$content_file" ]] || return 0

    # Warn about ignored agent-specific overrides
    if [[ -f "${skill_dir}/copilot.md" ]] || [[ -f "${skill_dir}/cursor.md" ]]; then
        log "WARNING: Agent-specific overrides (copilot.md/cursor.md) in '${skill_name}' are ignored — unified SKILL.md format is used"
    fi

    local target_dir=".agents/skills/${skill_name}"
    local target_file="${target_dir}/SKILL.md"

    mkdir -p "$target_dir"
    cp "$content_file" "$target_file"
    echo "$target_file" >> "${MANIFEST_FILE}.new"

    # Copy additional files (scripts/, references/, assets/, etc.)
    local src_path rel_path dest top
    while IFS= read -r src_path; do
        rel_path="${src_path#${skill_dir}/}"
        top="${rel_path%%/*}"
        case "$top" in
            SKILL.md|copilot.md|cursor.md) continue ;;
        esac
        dest="${target_dir}/${rel_path}"
        mkdir -p "$(dirname "$dest")"
        cp "$src_path" "$dest"
        echo "$dest" >> "${MANIFEST_FILE}.new"
        log_verbose "    + $dest"
    done < <(find "$skill_dir" -mindepth 1 -type f | sort)

    log_verbose "  Skill: $target_dir/"
}

# Install a command for GitHub Copilot.
# Creates .github/instructions/_shared.<name>.instructions.md
install_command_for_copilot() {
    local cmd_name="$1"
    local cmd_dir="$2"

    # Prefer agent-specific override, fall back to generic
    local content_file="${cmd_dir}/copilot.prompt.md"
    [[ -f "$content_file" ]] || content_file="${cmd_dir}/prompt.md"
    [[ -f "$content_file" ]] || return 0

    local target_dir=".github/instructions"
    local target_file="${target_dir}/${SHARED_PREFIX}.${cmd_name}.instructions.md"

    mkdir -p "$target_dir"
    cp "$content_file" "$target_file"
    echo "$target_file" >> "${MANIFEST_FILE}.new"
    log_verbose "  Copilot instruction: $target_file"
}

# Install a command for Cursor.
# Creates .cursor/commands/_shared.<name>.md
install_command_for_cursor() {
    local cmd_name="$1"
    local cmd_dir="$2"

    # Prefer agent-specific override, fall back to generic
    local content_file="${cmd_dir}/cursor.md"
    [[ -f "$content_file" ]] || content_file="${cmd_dir}/prompt.md"
    [[ -f "$content_file" ]] || return 0

    local target_dir=".cursor/commands"
    local target_file="${target_dir}/${SHARED_PREFIX}.${cmd_name}.md"

    mkdir -p "$target_dir"
    cp "$content_file" "$target_file"
    echo "$target_file" >> "${MANIFEST_FILE}.new"
    log_verbose "  Cursor command:  $target_file"
}

# ---------------------------------------------------------------------------
# Cleanup
# ---------------------------------------------------------------------------

# Remove files that were in the previous manifest but not in the new one.
# Also recursively removes empty parent directories up to .agents/skills/.
cleanup_stale_files() {
    [[ -f "$MANIFEST_FILE" ]] || return 0

    while IFS= read -r file; do
        [[ -z "$file" ]] && continue
        if ! grep -qxF "$file" "${MANIFEST_FILE}.new" 2>/dev/null; then
            if [[ -f "$file" ]]; then
                rm -f "$file"
                log "Removed stale: $file"
                # Prune empty parent directories up to (not including) .agents/skills/
                local parent
                parent="$(dirname "$file")"
                while [[ "$parent" != "." && "$parent" != ".agents/skills" && "$parent" != ".agents" ]]; do
                    if [[ -d "$parent" ]] && [[ -z "$(ls -A "$parent" 2>/dev/null)" ]]; then
                        rmdir "$parent" 2>/dev/null && log "Removed empty directory: $parent"
                        parent="$(dirname "$parent")"
                    else
                        break
                    fi
                done
            fi
        fi
    done < "$MANIFEST_FILE"
}

# Remove cached archives and extractions for tags no longer in use.
cleanup_old_cache() {
    local current_tags="$1"

    if [[ -d "${STATE_DIR}/extracted" ]]; then
        for dir in "${STATE_DIR}/extracted"/*/; do
            [[ -d "$dir" ]] || continue
            local tag
            tag="$(basename "$dir")"
            if ! echo "$current_tags" | grep -qxF "$tag"; then
                rm -rf "$dir"
                log_verbose "Cleaned cache: extracted/$tag"
            fi
        done
    fi

    if [[ -d "${STATE_DIR}/archive" ]]; then
        for f in "${STATE_DIR}/archive"/*.tar.gz; do
            [[ -f "$f" ]] || continue
            local tag
            tag="$(basename "$f" .tar.gz)"
            if ! echo "$current_tags" | grep -qxF "$tag"; then
                rm -f "$f"
                log_verbose "Cleaned cache: archive/${tag}.tar.gz"
            fi
        done
    fi
}

# ---------------------------------------------------------------------------
# Locking
# ---------------------------------------------------------------------------

acquire_lock() {
    mkdir -p "$STATE_DIR"

    if ! mkdir "$LOCK_DIR" 2>/dev/null; then
        # Check whether the lock is stale
        if [[ -f "${LOCK_DIR}/pid" ]]; then
            local lock_info lock_pid lock_time current_time
            lock_info="$(cat "${LOCK_DIR}/pid" 2>/dev/null)" || true
            lock_pid="$(echo "$lock_info" | awk '{print $1}')"
            lock_time="$(echo "$lock_info" | awk '{print $2}')"
            current_time="$(date +%s)"

            # Lock is stale if: process is dead, OR lock is older than 5 minutes
            local stale=false
            if [[ -n "$lock_pid" ]] && ! kill -0 "$lock_pid" 2>/dev/null; then
                stale=true
            elif [[ -n "$lock_time" ]] && [[ -n "$current_time" ]] \
                 && [[ $((current_time - lock_time)) -gt 300 ]]; then
                stale=true
            fi

            if [[ "$stale" == true ]]; then
                rm -rf "$LOCK_DIR"
                mkdir "$LOCK_DIR" 2>/dev/null || { log "Cannot acquire lock, skipping"; exit 0; }
            else
                log "Another update is running (PID ${lock_pid:-?}), skipping"
                exit 0
            fi
        else
            log "Another update is running, skipping"
            exit 0
        fi
    fi

    # Write PID and timestamp
    echo "$$ $(date +%s)" > "${LOCK_DIR}/pid"
    trap 'rm -rf "$LOCK_DIR" 2>/dev/null' EXIT INT TERM
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

main() {
    # Navigate to repo root
    local repo_root
    repo_root="$(git rev-parse --show-toplevel 2>/dev/null)" \
        || die "Not inside a git repository"
    cd "$repo_root"

    # Check for config file
    if [[ ! -f "$CONFIG_FILE" ]]; then
        log_verbose "No $CONFIG_FILE found, nothing to do"
        exit 0
    fi

    # Install git hooks (idempotent — safe to run before lock)
    install_hooks

    # Set up background logging
    if [[ "$BACKGROUND" == true ]]; then
        mkdir -p "$STATE_DIR"
        : > "$LOG_FILE"
    fi

    # Acquire lock (prevents concurrent runs from git hooks)
    acquire_lock

    # Parse config
    parse_config

    log "Updating from ${PROJECT_PATH}..."

    # ---- Resolve versions ----

    local latest_tag=""

    # Check if any unpinned entries need "latest" resolution
    local needs_latest=false
    if [[ ${#SKILLS[@]} -gt 0 ]]; then
        for entry in "${SKILLS[@]}"; do
            if [[ "$entry" != *@* ]]; then needs_latest=true; break; fi
        done
    fi
    if [[ "$needs_latest" != true ]] && [[ ${#COMMANDS[@]} -gt 0 ]]; then
        for entry in "${COMMANDS[@]}"; do
            if [[ "$entry" != *@* ]]; then needs_latest=true; break; fi
        done
    fi

    if [[ "$needs_latest" == true ]]; then
        if latest_tag="$(resolve_latest_tag)"; then
            log "Latest release: $latest_tag"
        else
            log "WARNING: Cannot reach GitLab to resolve latest version"
            if [[ -f "$INSTALLED_FILE" ]]; then
                log "Using cached data"
                exit 0
            else
                die "No cached data and cannot reach GitLab"
            fi
        fi
    fi

    # Resolve skill versions
    if [[ ${#SKILLS[@]} -gt 0 ]]; then
        for entry in "${SKILLS[@]}"; do
            local name tag
            if [[ "$entry" == *@* ]]; then
                name="${entry%%@*}"; tag="${entry#*@}"
            else
                name="$entry"; tag="$latest_tag"
            fi
            SKILL_VERSIONS+=("${name}:${tag}")
        done
    fi

    # Resolve command versions
    if [[ ${#COMMANDS[@]} -gt 0 ]]; then
        for entry in "${COMMANDS[@]}"; do
            local name tag
            if [[ "$entry" == *@* ]]; then
                name="${entry%%@*}"; tag="${entry#*@}"
            else
                name="$entry"; tag="$latest_tag"
            fi
            COMMAND_VERSIONS+=("${name}:${tag}")
        done
    fi

    # ---- Check if update is needed ----

    local new_state="format:${INSTALL_FORMAT_VERSION}"$'\n'
    if [[ ${#SKILL_VERSIONS[@]} -gt 0 ]]; then
        for sv in "${SKILL_VERSIONS[@]}"; do new_state+="${sv}"$'\n'; done
    fi
    if [[ ${#COMMAND_VERSIONS[@]} -gt 0 ]]; then
        for cv in "${COMMAND_VERSIONS[@]}"; do new_state+="command:${cv}"$'\n'; done
    fi
    new_state="$(printf '%s' "$new_state" | sort)"

    local current_state=""
    if [[ -f "$INSTALLED_FILE" ]]; then
        current_state="$(cat "$INSTALLED_FILE" 2>/dev/null)" || true
    fi

    if [[ "$current_state" == "$new_state" ]] && [[ "$FORCE" != true ]]; then
        log "Already up to date"
        exit 0
    fi

    # ---- Download & extract ----

    local all_entries=""
    if [[ ${#SKILL_VERSIONS[@]} -gt 0 ]]; then
        for sv in "${SKILL_VERSIONS[@]}"; do all_entries+="${sv}"$'\n'; done
    fi
    if [[ ${#COMMAND_VERSIONS[@]} -gt 0 ]]; then
        for cv in "${COMMAND_VERSIONS[@]}"; do all_entries+="${cv}"$'\n'; done
    fi
    local unique_tags
    unique_tags="$(printf '%s' "$all_entries" | cut -d: -f2 | sort -u)"

    for tag in $unique_tags; do
        if ! download_archive "$tag"; then
            if [[ -f "$INSTALLED_FILE" ]]; then
                log "WARNING: Download failed for $tag — keeping cached skills"
                exit 0
            fi
            die "Failed to download archive for $tag and no cache available"
        fi
        extract_archive "$tag"
    done

    # ---- Install skills ----

    rm -f "${MANIFEST_FILE}.new"
    touch "${MANIFEST_FILE}.new"

    if [[ ${#SKILL_VERSIONS[@]} -gt 0 ]]; then
        for sv in "${SKILL_VERSIONS[@]}"; do
            local name="${sv%%:*}"
            local tag="${sv#*:}"
            local skill_dir="${STATE_DIR}/extracted/${tag}/skills/${name}"

            if [[ ! -d "$skill_dir" ]]; then
                log "WARNING: Skill '${name}' not found in release ${tag}, skipping"
                continue
            fi

            log "Installing skill: ${name} (${tag})"
            install_skill "$name" "$skill_dir"
        done
    fi

    # ---- Install commands ----

    if [[ ${#COMMAND_VERSIONS[@]} -gt 0 ]]; then
        for cv in "${COMMAND_VERSIONS[@]}"; do
            local name="${cv%%:*}"
            local tag="${cv#*:}"
            local cmd_dir="${STATE_DIR}/extracted/${tag}/commands/${name}"

            if [[ ! -d "$cmd_dir" ]]; then
                log "WARNING: Command '${name}' not found in release ${tag}, skipping"
                continue
            fi

            log "Installing command: ${name} (${tag})"
            install_command_for_copilot "$name" "$cmd_dir"
            install_command_for_cursor "$name" "$cmd_dir"
        done
    fi

    # ---- Cleanup ----

    cleanup_stale_files
    cleanup_old_cache "$unique_tags"

    # ---- Commit new state ----

    mv "${MANIFEST_FILE}.new" "$MANIFEST_FILE"
    printf '%s\n' "$new_state" > "$INSTALLED_FILE"

    local skill_count=0 cmd_count=0
    if [[ ${#SKILL_VERSIONS[@]} -gt 0 ]]; then
        for _ in "${SKILL_VERSIONS[@]}"; do skill_count=$((skill_count + 1)); done
    fi
    if [[ ${#COMMAND_VERSIONS[@]} -gt 0 ]]; then
        for _ in "${COMMAND_VERSIONS[@]}"; do cmd_count=$((cmd_count + 1)); done
    fi
    log "Done — ${skill_count} skill(s), ${cmd_count} command(s) installed"
}

main "$@"
