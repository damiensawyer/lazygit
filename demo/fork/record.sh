#!/bin/sh
#
# Records the GIFs of this fork's features that the README shows. Each one is
# a demo integration test (pkg/integration/tests/demo), run with --slow in a
# terminal that vhs records. Everything happens locally: the tests make their
# own repos and config, and the GIFs land in demo/fork/gifs.
#
# Usage: demo/fork/record.sh [demo name...]
# e.g.   demo/fork/record.sh browse_files overview_mode
#
# With no names, records all of the fork's demos. Any other demo test can be
# named too, which is handy for checking how a change looks in a recording.
#
# Needs vhs (https://github.com/charmbracelet/vhs), which needs ttyd and
# ffmpeg, and a Nerd Font. The detailed commit graph is drawn with the font in
# demo/fonts, which this script makes available to the recording itself, so it
# needn't be installed.

set -e

DEMOS="browse_files final_version full_file_diff github_actions keybindings_search overview_mode side_panel_width"

# The non-Mono Nerd Font variant; the Mono variant renders the icons too small
FONT=${DEMO_FONT:-CaskaydiaMono Nerd Font}

DIR=$(cd "$(dirname "$0")" && pwd)
ROOT=$(cd "$DIR/../.." && pwd)
OUT=$DIR/gifs

for TOOL in vhs ttyd ffmpeg go
do
    if ! command -v "$TOOL" > /dev/null 2>&1
    then
        echo "$TOOL could not be found" >&2
        exit 1
    fi
done

if [ "$#" -eq 0 ]
then
    # shellcheck disable=SC2086
    set -- $DEMOS
fi

cd "$ROOT"

# Add the demos to the tests list (if missing) so that they can be run
go generate pkg/integration/tests/tests.go

SCRATCH=$(mktemp -d)
trap 'rm -rf "$SCRATCH"' EXIT

cat > "$SCRATCH/fonts.conf" <<EOF
<?xml version="1.0"?>
<!DOCTYPE fontconfig SYSTEM "fonts.dtd">
<fontconfig>
  <include ignore_missing="yes">/etc/fonts/fonts.conf</include>
  <dir>$ROOT/demo/fonts</dir>
  <cachedir>$SCRATCH/fontcache</cachedir>
</fontconfig>
EOF

# The terminal that vhs records runs in a browser, which finds its fonts
# through fontconfig.
export FONTCONFIG_FILE="$SCRATCH/fonts.conf"

# The integration test runner builds lazygit into a fixed path under the temp
# directory, so a test run from another checkout at the same time would
# otherwise swap our binary for its own.
mkdir -p "$SCRATCH/tmp"
export TMPDIR="$SCRATCH/tmp"

mkdir -p "$OUT"

# integration_test_steps prints the steps of a tape that records the demo
# integration test $1.
#
# Recording starts once lazygit has drawn the top left corner of a view frame,
# and stops once the end marker shows that lazygit has exited. The two quotes in
# the marker keep it out of the command line, so that waiting for it cannot
# match the command that prints it.
integration_test_steps() {
    TEST=pkg/integration/tests/demo/$1.go

    if [ ! -f "$TEST" ]
    then
        echo "There is no demo called $1 ($TEST)" >&2
        exit 1
    fi

    cat <<EOF
Hide
Type "clear; go run cmd/integration_test/main.go cli --slow $TEST; echo VHS''DONE"
Enter
Wait+Screen@180s /╭/
Show
Wait+Screen@600s /VHSDONE/
Hide
EOF
}

for NAME in "$@"
do
    STEPS=$(integration_test_steps "$NAME")

    echo "Recording $NAME"

    TAPE="$SCRATCH/$NAME.tape"
    RECORDING="$SCRATCH/$NAME.mp4"

    # The demos are laid out for a 120x35 terminal, which is what these pixel
    # sizes produce at this font size and padding. If you change any of them,
    # record a tape that runs `stty size` and adjust them until the grid is
    # 120x35 again.
    cat > "$TAPE" <<EOF
Output "$RECORDING"

Set Shell bash
Set FontFamily "$FONT,Flog Symbols Demo"
Set FontSize 16
Set LineHeight 1.0
Set Padding 14
Set Width 1256
Set Height 704
Set TypingSpeed 0ms
Set Theme "Catppuccin Mocha"

$STEPS
EOF

    vhs "$TAPE" > /dev/null

    if [ ! -f "$RECORDING" ]
    then
        echo "vhs recorded $NAME but wrote no video." >&2
        echo "vhs 0.12.0 does this; see https://github.com/charmbracelet/vhs/issues/787." >&2
        exit 1
    fi

    # vhs keeps recording until the end marker reaches the screen, and by then
    # lazygit has exited and the terminal has been cleared. Find the first of
    # those blank frames, so that the GIF can end on the demo's last frame: a
    # cleared terminal has hardly any pixels that aren't background, while the
    # emptiest screen that lazygit draws still has the frames of its views.
    CUT=$(ffmpeg -v error -i "$RECORDING" \
        -vf "format=gray,lutyuv=y='if(gt(val\,60)\,255\,0)',signalstats,metadata=print:key=lavfi.signalstats.YAVG:file=-" \
        -an -f null - 2>/dev/null |
        LC_ALL=C awk -v blank=0.5 '
            /pts_time:/ { split($3, a, ":"); time = a[2] }
            /YAVG=/ { split($0, b, "="); n++; at[n] = time + 0; ink[n] = b[2] + 0 }
            END {
                k = n + 1
                while (k > 1 && ink[k - 1] < blank) {
                    k--
                }
                if (k <= n) {
                    printf "%.3f\n", at[k]
                }
            }')

    if [ -n "$CUT" ]
    then
        TRIM="-t $CUT"
    else
        TRIM=
    fi

    # A palette made for each GIF keeps its colours true to the recording
    # shellcheck disable=SC2086
    ffmpeg -y -loglevel error -i "$RECORDING" $TRIM \
        -vf "fps=15,split[a][b];[a]palettegen=stats_mode=diff[p];[b][p]paletteuse=dither=none:diff_mode=rectangle" \
        "$OUT/$NAME.gif"

    echo "Recorded $OUT/$NAME.gif"
done
