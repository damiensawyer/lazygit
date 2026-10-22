#!/bin/bash
#
# Records the GIFs of this fork's features that the README shows.
#
# Usage: demo/fork/record.sh [tape name...]
#
# Needs vhs (https://github.com/charmbracelet/vhs), which needs ttyd and
# ffmpeg, and a Nerd Font. The GitHub Actions demo reads the runs of
# jesseduffield/lazygit, so it needs to be logged in with `gh`.

set -e

DIR=$(cd "$(dirname "$0")" && pwd)
ROOT=$(cd "$DIR/../.." && pwd)
OUT=$DIR/gifs
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

FONT=${DEMO_FONT:-CaskaydiaMono Nerd Font Mono}

for cmd in vhs ttyd ffmpeg go gh; do
	if ! command -v "$cmd" >/dev/null; then
		echo "$cmd could not be found" >&2
		exit 1
	fi
done

(cd "$ROOT" && go build -o "$TMP/bin/lazygit" .)

mkdir -p "$TMP/config/lazygit" "$OUT"
cp "$DIR/config.yml" "$TMP/config/lazygit/config.yml"
printf '[user]\n\tname = Yang Wen-li\n\temail = yang@example.com\n[commit]\n\tgpgsign = false\n' >"$TMP/gitconfig"

# A repo whose origin has plenty of workflow runs to show
git init -q -b master "$TMP/ghrepo"
git -C "$TMP/ghrepo" remote add origin https://github.com/jesseduffield/lazygit.git
GIT_CONFIG_GLOBAL="$TMP/gitconfig" git -C "$TMP/ghrepo" commit -q --allow-empty -m init

cat >"$TMP/common.tape" <<EOF
Set Shell bash
Set FontFamily "$FONT"
Set FontSize 15
Set Width 1400
Set Height 800
Set Padding 12
Set Theme "Catppuccin Mocha"
Set TypingSpeed 80ms
Env XDG_CONFIG_HOME "$TMP/config"
Env XDG_STATE_HOME "$TMP/state"
Env GIT_CONFIG_GLOBAL "$TMP/gitconfig"
Env GH_TOKEN "$(gh auth token)"
Env PATH "$TMP/bin:/usr/bin:/bin"
EOF

if [ "$#" -eq 0 ]; then
	set -- $(cd "$DIR/tapes" && ls *.tape | sed 's/\.tape$//')
fi

for name in "$@"; do
	echo "Recording $name"
	sed -e "s#{{DIR}}#$DIR#g; s#{{TMP}}#$TMP#g; s#{{OUT}}#$OUT#g" \
		"$DIR/tapes/$name.tape" >"$TMP/$name.tape"
	vhs "$TMP/$name.tape" >/dev/null
done
