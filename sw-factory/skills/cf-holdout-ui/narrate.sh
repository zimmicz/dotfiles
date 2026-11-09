#!/usr/bin/env bash
set -euo pipefail
dir=$(realpath "$1") key=$2
font=/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf
work=$dir/narrate
rm -rf "$work"; mkdir -p "$work"
enc=(-c:v libvpx-vp9 -deadline realtime -cpu-used 8 -row-mt 1 -crf 40 -b:v 0 -pix_fmt yuv420p -r 25 -c:a libopus -ar 48000 -ac 1)

dur() { ffprobe -v error -show_entries format=duration -of csv=p=0 "$1"; }
max() { awk -v a="$1" -v b="$2" 'BEGIN{print (a>b)?a:b}'; }
add() { awk -v a="$1" -v b="$2" 'BEGIN{print a+b}'; }

speak() {
  local out=$1; shift
  local parts=() i=0 s
  for s in "$@"; do
    [[ -n $s ]] || continue
    i=$((i+1))
    uvx edge-tts@7.3.1 --voice cs-CZ-AntoninNeural --rate=+5% --text "$s" --write-media "$out.vo$i.mp3" >/dev/null 2>&1 && parts+=("$out.vo$i.mp3") || true
  done
  if ((${#parts[@]} == 0)); then
    ffmpeg -v error -y -f lavfi -i anullsrc=r=48000:cl=mono -t 0.1 "$out"
    return
  fi
  local in=() f="" n=0 p
  for p in "${parts[@]}"; do in+=(-i "$p"); f+="[$n]apad=pad_dur=0.4[a$n];"; n=$((n+1)); done
  for ((p=0; p<n; p++)); do f+="[a$p]"; done
  ffmpeg -v error -y "${in[@]}" -filter_complex "${f}concat=n=$n:v=0:a=1" -ar 48000 -ac 1 "$out"
}

card() {
  local id=$1 title=$2 audio=$3 out=$4
  printf '%s' "$title" | fold -s -w 38 > "$work/$id.title"
  local d; d=$(max "$(add "$(dur "$audio")" 0.3)" 2)
  ffmpeg -v error -y -f lavfi -i "color=c=0x1e293b:s=1280x720:d=$d:r=25" -i "$audio" \
    -vf "drawtext=fontfile=$font:textfile=$work/$id.title:expansion=none:fontcolor=white:fontsize=52:line_spacing=14:x=(w-tw)/2:y=(h-th)/2" \
    -af apad -t "$d" "${enc[@]}" "$out"
}

clip() {
  local src=$1 audio=$2 out=$3
  ffmpeg -v error -y -i "$src" -vf "mpdecimate=max=8,setpts=N/FRAME_RATE/TB,scale=1280:720:force_original_aspect_ratio=decrease,pad=1280:720:(ow-iw)/2:(oh-ih)/2" -an -r 25 -c:v libvpx-vp9 -deadline realtime -cpu-used 8 -row-mt 1 -crf 40 -b:v 0 "$out.v.webm"
  local v a d
  v=$(dur "$out.v.webm"); a=$(add "$(dur "$audio")" 0.5); d=$(max "$v" "$a")
  ffmpeg -v error -y -i "$out.v.webm" -i "$audio" \
    -vf "tpad=stop_mode=clone:stop_duration=$(awk -v d="$d" -v v="$v" 'BEGIN{print d-v}')" \
    -af apad -t "$d" "${enc[@]}" "$out"
}

list=$work/list.txt meta=$work/chapters.txt
: > "$list"; printf ';FFMETADATA1\n' > "$meta"
t=0
while IFS=$'\t' read -r -u 3 id title text; do
  [[ -n $id ]] || continue
  IFS='|' read -r -a s <<< "$text"
  if [[ -f $dir/$id.webm ]]; then speak "$work/$id.head.m4a" "${s[0]}"; else speak "$work/$id.head.m4a" "${s[@]}"; fi
  card "$id" "$title" "$work/$id.head.m4a" "$work/$id.card.webm"
  segs=("$work/$id.card.webm")
  if [[ -f $dir/$id.webm ]]; then
    speak "$work/$id.body.m4a" "${s[@]:1}"
    clip "$dir/$id.webm" "$work/$id.body.m4a" "$work/$id.clip.webm"
    segs+=("$work/$id.clip.webm")
  fi
  start=$t
  for g in "${segs[@]}"; do echo "file '$g'" >> "$list"; t=$(add "$t" "$(dur "$g")"); done
  printf '[CHAPTER]\nTIMEBASE=1/1000\nSTART=%d\nEND=%d\ntitle=%s\n' "$(awk -v x="$start" 'BEGIN{printf "%d", x*1000}')" "$(awk -v x="$t" 'BEGIN{printf "%d", x*1000}')" "$title" >> "$meta"
done 3< "$dir/narration.tsv"

ffmpeg -v error -y -f concat -safe 0 -i "$list" -i "$meta" -map 0 -map_chapters 1 -c copy "$dir/$key.short.webm"
echo "$dir/$key.short.webm"
