#!/usr/bin/env bash

# Simple Claude Code StatusLine Script
# Shows only context usage/progress information
# System-agnostic version using Python instead of jq
# add to your .claude/settings.json (I use it as global)
#
# {
#  …,
#  "statusLine": {
#    "type": "command",
#    "command": "/home/srb/.claude/simple-statusline.sh"
#  }
#  …,
# }



# Configure the progress bar ratio (1.0 = 100 chars, 0.5 = 50 chars, etc.)
# Lower values create a shorter, more compact progress bar
BAR_RATIO=0.5

# Single source of truth for color scheme
# Returns: bar_color and empty_block_color for given percentage
get_colors_for_percentage() {
    local pct=$1

    if [[ $pct -lt 10 ]]; then
        bar_color="\033[38;2;24;53;34m"     # RGB(24,53,34) - dark green
        empty_block_color="\033[38;5;235m"
    elif [[ $pct -lt 20 ]]; then
        bar_color="\033[38;2;21;62;33m"     # RGB(21,62,33) - dark green
        empty_block_color="\033[38;5;235m"
    elif [[ $pct -lt 30 ]]; then
        bar_color="\033[38;2;16;70;32m"     # RGB(16,70,32) - green
        empty_block_color="\033[38;5;235m"
    elif [[ $pct -lt 40 ]]; then
        bar_color="\033[38;2;11;78;28m"     # RGB(11,78,28) - green
        empty_block_color="\033[38;5;235m"
    elif [[ $pct -lt 50 ]]; then
        bar_color="\033[38;2;6;87;22m"      # RGB(6,87,22) - bright green
        empty_block_color="\033[38;5;235m"
    elif [[ $pct -lt 60 ]]; then
        bar_color="\033[38;2;46;89;0m"      # RGB(46,89,0) - yellow-green
        empty_block_color="\033[38;5;236m"
    elif [[ $pct -lt 70 ]]; then
        bar_color="\033[38;2;93;79;0m"      # RGB(93,79,0) - olive/yellow
        empty_block_color="\033[38;5;237m"
    elif [[ $pct -lt 80 ]]; then
        bar_color="\033[38;2;131;58;0m"     # RGB(131,58,0) - orange
        empty_block_color="\033[38;5;238m"
    elif [[ $pct -lt 90 ]]; then
        bar_color="\033[38;2;161;7;0m"      # RGB(161,7,0) - red-orange
        empty_block_color="\033[38;5;238m"
    else
        bar_color="\033[38;2;179;0;0m"      # RGB(179,0,0) - red
        empty_block_color="\033[38;5;238m"
    fi
}

# Get model badge color based on model name
get_model_color() {
    local model=$1

    if [[ "$model" == *"Sonnet"* ]]; then
        echo "\033[48;2;163;190;140m\033[1m\033[38;2;46;52;64m"  # a3be8c bg, 2e3440 text
    elif [[ "$model" == *"Opus"* ]]; then
        echo "\033[48;2;208;135;112m\033[1m\033[38;2;46;52;64m"  # d08770 bg, 2e3440 text
    elif [[ "$model" == *"Haiku"* ]]; then
        echo "\033[48;2;129;161;193m\033[1m\033[38;2;46;52;64m"  # 81a1c1 bg, 2e3440 text
    else
        echo "\033[48;2;216;222;233m\033[1m\033[38;2;216;222;233m"  # d8dee9 bg, d8dee9 text
    fi
}

# Center text in a fixed width field (16 characters)
center_text() {
    local text="$1"
    local width=16
    local text_len=${#text}
    local padding=$(( (width - text_len) / 2 ))
    local right_padding=$(( width - text_len - padding ))

    printf "%${padding}s%s%${right_padding}s" "" "$text" ""
}

# Get formatted cwd and git branch suffix
get_cwd_suffix() {
    local cwd="$1"

    if [[ -z "$cwd" ]]; then
        echo ""
        return
    fi

    # Shorten home directory to ~
    local cwd_short=$(python3 -c "import os; print('$cwd'.replace(os.path.expanduser('~'), '~', 1))")

    # Get git branch if in a git repo
    local git_branch=$(git -C "$cwd" branch --show-current 2>/dev/null)

    if [[ -n "$git_branch" ]]; then
        echo " ${cwd_short} [${git_branch}]"
    else
        echo " ${cwd_short}"
    fi
}

# Check for --show-scale argument
if [[ "$1" == "--show-scale" ]]; then
    show_scale() {
        local min=$1
        local max=$2
        local mode=$3
        local pct

        # Calculate percentage based on mode
        case "$mode" in
            min)
                pct=$min
                ;;
            max)
                pct=$max
                ;;
            mid)
                pct=$(((min + max) / 2))
                ;;
        esac

        local bar_length=$(python3 -c "print(int(100 * $BAR_RATIO + 0.5))")
        local filled_blocks=$(python3 -c "print(int($pct * $BAR_RATIO + 0.5))")
        local empty_blocks=$((bar_length - filled_blocks))

        # Get colors from single source of truth
        get_colors_for_percentage "$pct"

        local reset="\033[0m"
        local progress_bar="${bar_color}"

        for ((i=0; i<filled_blocks; i++)); do progress_bar+="█"; done
        progress_bar+="${empty_block_color}"
        for ((i=0; i<empty_blocks; i++)); do progress_bar+="█"; done
        progress_bar+="${reset}"

        printf "%3d-%3d%%: %b\n" "$min" "$max" "$progress_bar"
    }

    display_mode() {
        local mode=$1

        # Set header based on mode
        case "$mode" in
            min)
                echo "Color Scale Demo (showing range with minimum value):"
                ;;
            max)
                echo "Color Scale Demo (showing range with maximum value):"
                ;;
            mid)
                echo "Color Scale Demo (showing range with midpoint value):"
                ;;
        esac

        echo ""
        show_scale 0 9 "$mode"      # 0-9%
        show_scale 10 19 "$mode"    # 10-19%
        show_scale 20 29 "$mode"    # 20-29%
        show_scale 30 39 "$mode"    # 30-39%
        show_scale 40 49 "$mode"    # 40-49%
        show_scale 50 59 "$mode"    # 50-59%
        show_scale 60 69 "$mode"    # 60-69%
        show_scale 70 79 "$mode"    # 70-79%
        show_scale 80 89 "$mode"    # 80-89%
        show_scale 90 100 "$mode"   # 90-100%
    }

    show_single_bar() {
        local pct=$1
        local bar_length=$(python3 -c "print(int(100 * $BAR_RATIO + 0.5))")
        local filled_blocks=$(python3 -c "print(int($pct * $BAR_RATIO + 0.5))")
        local empty_blocks=$((bar_length - filled_blocks))

        # Get colors from single source of truth
        get_colors_for_percentage "$pct"

        local reset="\033[0m"
        local progress_bar="${bar_color}"

        for ((i=0; i<filled_blocks; i++)); do progress_bar+="█"; done
        progress_bar+="${empty_block_color}"
        for ((i=0; i<empty_blocks; i++)); do progress_bar+="█"; done
        progress_bar+="${reset}"

        printf "\r%3d%%: %b" "$pct" "$progress_bar"
    }

    # Check if mode argument is provided
    if [[ -n "$2" ]]; then
        # Validate mode
        if [[ "$2" != "min" && "$2" != "max" && "$2" != "mid" && "$2" != "animate" ]]; then
            echo "Error: Invalid argument '$2'. Use: min, max, mid, or animate"
            exit 1
        fi

        # Handle animate mode
        if [[ "$2" == "animate" ]]; then
            trap 'echo ""; exit 0' INT  # Allow Ctrl+C to exit gracefully with newline
            while true; do
                for pct in {0..100}; do
                    show_single_bar "$pct"
                    sleep 0.1
                done
                sleep 0.5  # Brief pause at 100% before restarting
            done
        else
            # Show specific mode once
            display_mode "$2"
        fi
    else
        # No argument - animate by default
        trap 'echo ""; exit 0' INT  # Allow Ctrl+C to exit gracefully with newline
        while true; do
            for pct in {0..100}; do
                show_single_bar "$pct"
                sleep 0.2
            done
            sleep 0.5  # Brief pause at 100% before restarting
        done
    fi
    exit 0
fi

# Read JSON input from stdin and parse once
input=$(cat)

# Parse JSON once and extract all needed fields using a unique delimiter
IFS='|' read -r model_name transcript_path cwd <<< $(echo "$input" | python3 -c "
import sys, json
data = json.load(sys.stdin)
model = data.get('model', {}).get('display_name', 'Claude')
transcript = data.get('transcript_path', '')
cwd = data.get('cwd', '')
print(f'{model}|{transcript}|{cwd}')
")

# Function to calculate context breakdown and progress
calculate_context() {
    # Determine usable context limit
    if [[ "$model_name" == *"[1m]"* ]]; then
        context_limit=800000
    elif [[ "$model_name" == *"Sonnet"* ]]; then
        context_limit=200000
    else
        context_limit=200000
    fi

    if [[ -n "$transcript_path" && -f "$transcript_path" ]]; then
        # Parse transcript to get real token usage
        result=$(python3 -c "
import sys, json

try:
    with open('$transcript_path', 'r', encoding='utf-8', errors='backslashreplace') as f:
        lines = f.readlines()

    most_recent_usage = None
    most_recent_timestamp = None

    # Parse all lines to find most recent usage
    for line in lines:
        try:
            data = json.loads(line.strip())
            # Skip sidechain entries (subagent calls)
            if data.get('isSidechain', False):
                continue

            # Check for usage data in main-chain messages
            if data.get('message', {}).get('usage'):
                timestamp = data.get('timestamp')
                if timestamp and (not most_recent_timestamp or timestamp > most_recent_timestamp):
                    most_recent_timestamp = timestamp
                    most_recent_usage = data['message']['usage']
        except:
            continue

    # Calculate context length (input + cache tokens only, NOT output)
    if most_recent_usage:
        context_length = (
            most_recent_usage.get('input_tokens', 0) +
            most_recent_usage.get('cache_read_input_tokens', 0) +
            most_recent_usage.get('cache_creation_input_tokens', 0)
        )
        print(f'{context_length}')
    else:
        print('0')
except Exception as e:
    print('0')
" 2>/dev/null)

        total_tokens="$result"

        # Check if no usage data is available yet
        if [[ "$total_tokens" == "0" ]]; then
            model_color=$(get_model_color "$model_name")
            suffix=$(get_cwd_suffix "$cwd")
            echo -e "${model_color}$(center_text "$model_name")\033[0m \033[38;5;250mcontext size N/A${suffix}\033[0m"
            return
        fi

        # Calculate actual context usage percentage (capped at 100%)
        if [[ $total_tokens -gt 0 ]]; then
            progress_pct_int=$(python3 -c "print(min(100, int($total_tokens * 100 / $context_limit)))")
        else
            progress_pct_int=0
        fi
    else
        # Default values when no transcript available
        total_tokens=0
        progress_pct_int=0
    fi

    # Format token count in 'k' format
    formatted_tokens=$(python3 -c "print(f'{$total_tokens // 1000}k')")
    formatted_limit=$(python3 -c "print(f'{$context_limit // 1000}k')")

    # Create progress bar
    bar_length=$(python3 -c "print(int(100 * $BAR_RATIO + 0.5))")
    filled_blocks=$(python3 -c "print(int($progress_pct_int * $BAR_RATIO + 0.5))")
    empty_blocks=$((bar_length - filled_blocks))

    # Build progress bar with colors
    get_colors_for_percentage "$progress_pct_int"
    model_color=$(get_model_color "$model_name")
    text_color="\033[38;5;250m"     # BFBDB6 light gray
    reset="\033[0m"

    progress_bar="${model_color}$(center_text "$model_name")${reset}${bar_color}"
    for ((i=0; i<filled_blocks; i++)); do progress_bar+="█"; done
    progress_bar+="${empty_block_color}"
    for ((i=0; i<empty_blocks; i++)); do progress_bar+="█"; done
    progress_bar+="${reset}${text_color} ${progress_pct_int}% (${formatted_tokens}/${formatted_limit})"
    progress_bar+="$(get_cwd_suffix "$cwd")"

    echo -e "$progress_bar"
}

# Output just the context information
calculate_context
