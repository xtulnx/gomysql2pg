#!/usr/bin/env bash
# 顺序迁移 configs/ 下所有 yml，每对独立日志，单对失败不中断后续。
# 用法: bash run_batch.sh [config_dir]

CONFIG_DIR="${1:-configs}"
BINARY="./gomysql2pg"
TS="$(date +%Y%m%d-%H%M%S)"
BATCH_LOG="batch-${TS}.log"

# 从 yml 中读取 section.key（section=src|dest，只支持二层缩进、无嵌套 map）
yml_get() {
    local file="$1" section="$2" key="$3"
    awk -v sec="$section" -v k="$key" '
        /^[A-Za-z_][A-Za-z0-9_]*:[[:space:]]*$/ {
            cur = $0; sub(/:.*/, "", cur)
            in_sec = (cur == sec)
            next
        }
        /^[^[:space:]]/ { in_sec = 0; next }
        in_sec {
            line = $0
            sub(/^[[:space:]]+/, "", line)
            if (match(line, /^[A-Za-z_][A-Za-z0-9_]*:/)) {
                name = substr(line, 1, RLENGTH - 1)
                if (name == k) {
                    val = substr(line, RLENGTH + 1)
                    sub(/^[[:space:]]+/, "", val)
                    sub(/[[:space:]]+$/, "", val)
                    gsub(/^"|"$/, "", val)
                    gsub(/^'\''|'\''$/, "", val)
                    print val
                    exit
                }
            }
        }
    ' "$file"
}

if [[ ! -x "$BINARY" ]]; then
    echo "binary not found or not executable: $BINARY" >&2
    exit 2
fi
if [[ ! -d "$CONFIG_DIR" ]]; then
    echo "config dir not found: $CONFIG_DIR" >&2
    exit 2
fi

shopt -s nullglob
files=("$CONFIG_DIR"/*.yml "$CONFIG_DIR"/*.yaml)
if [[ ${#files[@]} -eq 0 ]]; then
    echo "no yml files under $CONFIG_DIR" >&2
    exit 2
fi
IFS=$'\n' files=($(printf '%s\n' "${files[@]}" | sort))
unset IFS

# 预览待迁移的配置文件
echo "Found ${#files[@]} config(s) to migrate:"
i=1
for f in "${files[@]}"; do
    s_host=$(yml_get "$f" src  host)
    s_port=$(yml_get "$f" src  port)
    s_db=$(  yml_get "$f" src  database)
    d_host=$(yml_get "$f" dest host)
    d_port=$(yml_get "$f" dest port)
    d_db=$(  yml_get "$f" dest database)
    d_user=$(yml_get "$f" dest username)
    printf "  [%d] %s  |  src=%s:%s/%s  ->  dest=%s@%s:%s/%s\n" \
        "$i" "$f" \
        "${s_host:-?}" "${s_port:-?}" "${s_db:-?}" \
        "${d_user:-?}" "${d_host:-?}" "${d_port:-?}" "${d_db:-?}"
    i=$((i + 1))
done
echo

# 交互式确认
read -r -p "Proceed with these ${#files[@]} migration(s)? [yes/no]: " answer
answer=$(printf '%s' "$answer" | tr '[:upper:]' '[:lower:]')
case "$answer" in
    yes|y) ;;
    *) echo "aborted by user"; exit 1 ;;
esac

succ=()
fail=()

{
    echo "=== batch start $(date '+%Y-%m-%d %H:%M:%S') total=${#files[@]} ==="
    for f in "${files[@]}"; do
        echo
        echo "============================================================"
        echo ">>> [$(date '+%H:%M:%S')] running: $f"
        echo "============================================================"
        "$BINARY" --config "$f"
        rc=$?
        if [[ $rc -eq 0 ]]; then
            echo "<<< [OK] $f"
            succ+=("$f")
        else
            echo "<<< [FAIL rc=$rc] $f"
            fail+=("$f")
        fi
    done

    echo
    echo "=== batch done $(date '+%Y-%m-%d %H:%M:%S') ==="
    echo "success: ${#succ[@]}"
    for f in "${succ[@]}"; do echo "  OK   $f"; done
    echo "failed:  ${#fail[@]}"
    for f in "${fail[@]}"; do echo "  FAIL $f"; done
} 2>&1 | tee "$BATCH_LOG"

exit ${#fail[@]}
