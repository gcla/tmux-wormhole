#!/usr/bin/env bash

set -Eu -o pipefail

# from tmux-thumbs
function get-opt-value() {
  tmux show -vg "@wormhole-${1}" 2> /dev/null
}

# e.g. join_by , a b c => a,b,c
function join_by {
    local IFS="$1"
    shift
    echo "$*"
}

# I want a short token because I use this in a tmux window name that might be displayed
# in the status bar. I want it to be relatively unobtrusive.
#
# From https://stackoverflow.com/a/32484733
function random_token() {
    chars=abcdefghjkmnpqrstuvwxyzABCDEFGHJKLMNPQRTUVWXYZ2346789
    for i in {1..3} ; do
	printf "${chars:RANDOM%${#chars}:1}"
    done
}

# From https://stackoverflow.com/a/36490592
function tmp_dir() {
    dirname $(mktemp -u -t tmp.XXXXXXXXXX)
}

# This is run inside the tmux pane that is used in place of the one from
# which the plugin is launched. This just cats the saved pane contents, so
# it looks like the original... macOS doesn't have sleep infinity
function pane_command() {
    printf 'bash -c '\''cat "%s" ; sleep 100000'\' "${TMUX_WORMHOLE_TMP_FILE}"
}

######################################################################

TMUX_WORMHOLE_BIN="$HOME/.tmux/plugins/tmux-wormhole/tmux-wormhole"
if [[ ! -e "${TMUX_WORMHOLE_BIN}" ]] ; then
    tmux split-window "echo Could not execute tmux-wormhole binary $TMUX_WORMHOLE_BIN. Please check plugin installation. ; read"
    exit 1
fi

set -e

# Make sure every variable exists
TMUX_WORMHOLE_SAVE_FOLDER="$(get-opt-value save-folder)"
TMUX_WORMHOLE_OPEN_CMD="$(get-opt-value open-cmd)"
TMUX_WORMHOLE_NO_DEFAULT_OPEN="$(get-opt-value no-default-open)"
TMUX_WORMHOLE_NO_ASK_TO_OPEN="$(get-opt-value no-ask-to-open)"
TMUX_WORMHOLE_CAN_OVERWRITE="$(get-opt-value can-overwrite)"

# e.g. abc
TMUX_WORMHOLE_CURRENT="$(random_token)"

# e.g. wormhole-abc
TMUX_WORMHOLE_SESSION="wormhole-${TMUX_WORMHOLE_CURRENT}"

# e.g. /tmp/,tmux-wormhole-abc
TMUX_WORMHOLE_TMP_FILE="$(tmp_dir)/.tmux-wormhole-${TMUX_WORMHOLE_CURRENT}"

# Used for constructing a regex that matches a wormhole code. The PGP word list.
TMUX_WORMHOLE_PGP_WORD_LIST=(
    "aardvark" \
	"absurd" \
	"accrue" \
	"acme" \
	"adrift" \
	"adult" \
	"afflict" \
	"ahead" \
	"aimless" \
	"algol" \
	"allow" \
	"alone" \
	"ammo" \
	"ancient" \
	"apple" \
	"artist" \
	"assume" \
	"athens" \
	"atlas" \
	"aztec" \
	"baboon" \
	"backfield" \
	"backward" \
	"banjo" \
	"beaming" \
	"bedlamp" \
	"beehive" \
	"beeswax" \
	"befriend" \
	"belfast" \
	"berserk" \
	"billiard" \
	"bison" \
	"blackjack" \
	"blockade" \
	"blowtorch" \
	"bluebird" \
	"bombast" \
	"bookshelf" \
	"brackish" \
	"breadline" \
	"breakup" \
	"brickyard" \
	"briefcase" \
	"burbank" \
	"button" \
	"buzzard" \
	"cement" \
	"chairlift" \
	"chatter" \
	"checkup" \
	"chisel" \
	"choking" \
	"chopper" \
	"christmas" \
	"clamshell" \
	"classic" \
	"classroom" \
	"cleanup" \
	"clockwork" \
	"cobra" \
	"commence" \
	"concert" \
	"cowbell" \
	"crackdown" \
	"cranky" \
	"crowfoot" \
	"crucial" \
	"crumpled" \
	"crusade" \
	"cubic" \
	"dashboard" \
	"deadbolt" \
	"deckhand" \
	"dogsled" \
	"dragnet" \
	"drainage" \
	"dreadful" \
	"drifter" \
	"dropper" \
	"drumbeat" \
	"drunken" \
	"dupont" \
	"dwelling" \
	"eating" \
	"edict" \
	"egghead" \
	"eightball" \
	"endorse" \
	"endow" \
	"enlist" \
	"erase" \
	"escape" \
	"exceed" \
	"eyeglass" \
	"eyetooth" \
	"facial" \
	"fallout" \
	"flagpole" \
	"flatfoot" \
	"flytrap" \
	"fracture" \
	"framework" \
	"freedom" \
	"frighten" \
	"gazelle" \
	"geiger" \
	"glitter" \
	"glucose" \
	"goggles" \
	"goldfish" \
	"gremlin" \
	"guidance" \
	"hamlet" \
	"highchair" \
	"hockey" \
	"indoors" \
	"indulge" \
	"inverse" \
	"involve" \
	"island" \
	"jawbone" \
	"keyboard" \
	"kickoff" \
	"kiwi" \
	"klaxon" \
	"locale" \
	"lockup" \
	"merit" \
	"minnow" \
	"miser" \
	"mohawk" \
	"mural" \
	"music" \
	"necklace" \
	"neptune" \
	"newborn" \
	"nightbird" \
	"oakland" \
	"obtuse" \
	"offload" \
	"optic" \
	"orca" \
	"payday" \
	"peachy" \
	"pheasant" \
	"physique" \
	"playhouse" \
	"pluto" \
	"preclude" \
	"prefer" \
	"preshrunk" \
	"printer" \
	"prowler" \
	"pupil" \
	"puppy" \
	"python" \
	"quadrant" \
	"quiver" \
	"quota" \
	"ragtime" \
	"ratchet" \
	"rebirth" \
	"reform" \
	"regain" \
	"reindeer" \
	"rematch" \
	"repay" \
	"retouch" \
	"revenge" \
	"reward" \
	"rhythm" \
	"ribcage" \
	"ringbolt" \
	"robust" \
	"rocker" \
	"ruffled" \
	"sailboat" \
	"sawdust" \
	"scallion" \
	"scenic" \
	"scorecard" \
	"scotland" \
	"seabird" \
	"select" \
	"sentence" \
	"shadow" \
	"shamrock" \
	"showgirl" \
	"skullcap" \
	"skydive" \
	"slingshot" \
	"slowdown" \
	"snapline" \
	"snapshot" \
	"snowcap" \
	"snowslide" \
	"solo" \
	"southward" \
	"soybean" \
	"spaniel" \
	"spearhead" \
	"spellbind" \
	"spheroid" \
	"spigot" \
	"spindle" \
	"spyglass" \
	"stagehand" \
	"stagnate" \
	"stairway" \
	"standard" \
	"stapler" \
	"steamship" \
	"sterling" \
	"stockman" \
	"stopwatch" \
	"stormy" \
	"sugar" \
	"surmount" \
	"suspense" \
	"sweatband" \
	"swelter" \
	"tactics" \
	"talon" \
	"tapeworm" \
	"tempest" \
	"tiger" \
	"tissue" \
	"tonic" \
	"topmost" \
	"tracker" \
	"transit" \
	"trauma" \
	"treadmill" \
	"trojan" \
	"trouble" \
	"tumor" \
	"tunnel" \
	"tycoon" \
	"uncut" \
	"unearth" \
	"unwind" \
	"uproot" \
	"upset" \
	"upshot" \
	"vapor" \
	"village" \
	"virus" \
	"vulcan" \
	"waffle" \
	"wallet" \
	"watchword" \
	"wayside" \
	"willow" \
	"woodlark" \
	"zulu" \
	"adroitness" \
	"adviser" \
	"aftermath" \
	"aggregate" \
	"alkali" \
	"almighty" \
	"amulet" \
	"amusement" \
	"antenna" \
	"applicant" \
	"apollo" \
	"armistice" \
	"article" \
	"asteroid" \
	"atlantic" \
	"atmosphere" \
	"autopsy" \
	"babylon" \
	"backwater" \
	"barbecue" \
	"belowground" \
	"bifocals" \
	"bodyguard" \
	"bookseller" \
	"borderline" \
	"bottomless" \
	"bradbury" \
	"bravado" \
	"brazilian" \
	"breakaway" \
	"burlington" \
	"businessman" \
	"butterfat" \
	"camelot" \
	"candidate" \
	"cannonball" \
	"capricorn" \
	"caravan" \
	"caretaker" \
	"celebrate" \
	"cellulose" \
	"certify" \
	"chambermaid" \
	"cherokee" \
	"chicago" \
	"clergyman" \
	"coherence" \
	"combustion" \
	"commando" \
	"company" \
	"component" \
	"concurrent" \
	"confidence" \
	"conformist" \
	"congregate" \
	"consensus" \
	"consulting" \
	"corporate" \
	"corrosion" \
	"councilman" \
	"crossover" \
	"crucifix" \
	"cumbersome" \
	"customer" \
	"dakota" \
	"decadence" \
	"december" \
	"decimal" \
	"designing" \
	"detector" \
	"detergent" \
	"determine" \
	"dictator" \
	"dinosaur" \
	"direction" \
	"disable" \
	"disbelief" \
	"disruptive" \
	"distortion" \
	"document" \
	"embezzle" \
	"enchanting" \
	"enrollment" \
	"enterprise" \
	"equation" \
	"equipment" \
	"escapade" \
	"eskimo" \
	"everyday" \
	"examine" \
	"existence" \
	"exodus" \
	"fascinate" \
	"filament" \
	"finicky" \
	"forever" \
	"fortitude" \
	"frequency" \
	"gadgetry" \
	"galveston" \
	"getaway" \
	"glossary" \
	"gossamer" \
	"graduate" \
	"gravity" \
	"guitarist" \
	"hamburger" \
	"hamilton" \
	"handiwork" \
	"hazardous" \
	"headwaters" \
	"hemisphere" \
	"hesitate" \
	"hideaway" \
	"holiness" \
	"hurricane" \
	"hydraulic" \
	"impartial" \
	"impetus" \
	"inception" \
	"indigo" \
	"inertia" \
	"infancy" \
	"inferno" \
	"informant" \
	"insincere" \
	"insurgent" \
	"integrate" \
	"intention" \
	"inventive" \
	"istanbul" \
	"jamaica" \
	"jupiter" \
	"leprosy" \
	"letterhead" \
	"liberty" \
	"maritime" \
	"matchmaker" \
	"maverick" \
	"medusa" \
	"megaton" \
	"microscope" \
	"microwave" \
	"midsummer" \
	"millionaire" \
	"miracle" \
	"misnomer" \
	"molasses" \
	"molecule" \
	"montana" \
	"monument" \
	"mosquito" \
	"narrative" \
	"nebula" \
	"newsletter" \
	"norwegian" \
	"october" \
	"ohio" \
	"onlooker" \
	"opulent" \
	"orlando" \
	"outfielder" \
	"pacific" \
	"pandemic" \
	"pandora" \
	"paperweight" \
	"paragon" \
	"paragraph" \
	"paramount" \
	"passenger" \
	"pedigree" \
	"pegasus" \
	"penetrate" \
	"perceptive" \
	"performance" \
	"pharmacy" \
	"phonetic" \
	"photograph" \
	"pioneer" \
	"pocketful" \
	"politeness" \
	"positive" \
	"potato" \
	"processor" \
	"provincial" \
	"proximate" \
	"puberty" \
	"publisher" \
	"pyramid" \
	"quantity" \
	"racketeer" \
	"rebellion" \
	"recipe" \
	"recover" \
	"repellent" \
	"replica" \
	"reproduce" \
	"resistor" \
	"responsive" \
	"retraction" \
	"retrieval" \
	"retrospect" \
	"revenue" \
	"revival" \
	"revolver" \
	"sandalwood" \
	"sardonic" \
	"saturday" \
	"savagery" \
	"scavenger" \
	"sensation" \
	"sociable" \
	"souvenir" \
	"specialist" \
	"speculate" \
	"stethoscope" \
	"stupendous" \
	"supportive" \
	"surrender" \
	"suspicious" \
	"sympathy" \
	"tambourine" \
	"telephone" \
	"therapist" \
	"tobacco" \
	"tolerance" \
	"tomorrow" \
	"torpedo" \
	"tradition" \
	"travesty" \
	"trombonist" \
	"truncated" \
	"typewriter" \
	"ultimate" \
	"undaunted" \
	"underfoot" \
	"unicorn" \
	"unify" \
	"universe" \
	"unravel" \
	"upcoming" \
	"vacancy" \
	"vagabond" \
	"vertigo" \
	"virginia" \
	"visitor" \
	"vocalist" \
	"voyager" \
	"warranty" \
	"waterloo" \
	"whimsical" \
	"wichita" \
	"wilmington" \
	"wyoming" \
	"yesteryear" \
	"yucatan" \
)

TMUX_WORMHOLE_PGP_RE=$(printf '\\b[[:digit:]]{1,3}(-(%s)){2,}' $(join_by '|' "${TMUX_WORMHOLE_PGP_WORD_LIST[@]}"))

# Capture the width and height so I can set up my fake tmux pane with the same dimensions
IFS=, read TID TWID THEI TZOOM DUMMY \
   <<<"$(tmux list-panes -F '#{pane_id},#{pane_width},#{pane_height},#{window_zoomed_flag},#{pane_active}' | grep ',1$')"

# Save the current pane's contents. I'll scrape this for the wormhole code, and also display
# this inside the gowid terminal which will attach to a dummy tmux session - the terminal
# inside that session will show these contents using cat > /dev/tty ; sleep
tmux capture-pane -e -p -J -t "${TID}" > "${TMUX_WORMHOLE_TMP_FILE}"

# Strip the last newline so we don't get an extra linefeed when displaying in the gowid terminal.
truncate -s -1 "${TMUX_WORMHOLE_TMP_FILE}"

# This is passed to the gowid program - so it knows what to show the user.
TMUX_WORMHOLE_CODE=$(grep -E -o -a "${TMUX_WORMHOLE_PGP_RE}" "${TMUX_WORMHOLE_TMP_FILE}" | tail -n 1)

# This session is used to construct a pane that looks like the current pane, but with the
# wormhole code highlighted. I put it under another socket so I don't have to worry about
# the active session. 
TMUX_WORMHOLE_SESSION_ID=$(tmux -L wormhole new-session -s "${TMUX_WORMHOLE_SESSION}" -P -F '#{session_id}' -d)

# This pane will be displayed inside a gowid terminal inside the current pane. Since the space
# available to the gowid terminal is exactly the same space available to the current pane, I
# need to turn off the status bar in the wormhole session so that when the gowid app runs
# tmux attach, they don't see their main status bar, then the wormhole session status bar too.
# This makes it look seamless.
tmux -L wormhole set -g status off

# Make sure the window holding the replacement pane is the right size, so there
# are no resize events affecting the layout
if [[ "$TZOOM" != "1" ]] ; then
    tmux -L wormhole resize-window -x "${TWID}" -y "${THEI}"
fi
    
# cat the pane contents through grep to highlight; then sleep. Now the fake pane
# is set up and ready to be displayed instead of the current one.
tmux -L wormhole respawn-window -k "$(pane_command)"

# Open a new window without changing focus. I will save the current pane over on that window so
# that I can restore it after the plugin runs.
#
# e.g. @685
TMUX_WORMHOLE_ORIG_WINDOW=$(tmux new-window -P -d -F '#{window_id}' -n ${TMUX_WORMHOLE_SESSION})

# Replace the current pane - which is a throaway swapped over from the tmp
# window above - with the plugin. The plugin will read the pane output saved
# above and load it in a gowid terminal, so it looks like the original pane...
# When the plugin ends, swap the original pane back to its original location,
# then make sure my wormhole tmux session with pane displaying the highlighted
# terminal contents is cleaned up.
tmux respawn-pane -k -t "${TMUX_WORMHOLE_ORIG_WINDOW}" \
     -e TMUX_WORMHOLE_CODE="${TMUX_WORMHOLE_CODE}" \
     -e TMUX_WORMHOLE_SESSION="${TMUX_WORMHOLE_SESSION}" \
     -e TMUX_WORMHOLE_SAVE_FOLDER="${TMUX_WORMHOLE_SAVE_FOLDER}" \
     -e TMUX_WORMHOLE_OPEN_CMD="${TMUX_WORMHOLE_OPEN_CMD}" \
     -e TMUX_WORMHOLE_NO_DEFAULT_OPEN="${TMUX_WORMHOLE_NO_DEFAULT_OPEN}" \
     -e TMUX_WORMHOLE_NO_ASK_TO_OPEN="${TMUX_WORMHOLE_NO_ASK_TO_OPEN}" \
     -e TMUX_WORMHOLE_CAN_OVERWRITE="${TMUX_WORMHOLE_CAN_OVERWRITE}" \
     /usr/bin/env bash -c "if ! $TMUX_WORMHOLE_BIN ; then echo Hit enter. ; read ; fi ; \
      tmux swap-pane -t \"${TMUX_WORMHOLE_ORIG_WINDOW}\" ; \
      [[ "$TZOOM" = "1" ]] && tmux resize-pane -Z ; \
      tmux -L wormhole kill-session -t \"${TMUX_WORMHOLE_SESSION}\" ; 
      rm -f \"${TMUX_WORMHOLE_TMP_FILE}\" "

# This gives the plugin a little time to startup, launch its own terminal, etc
# and avoids flicker.
# sleep 0.2s

# Save current pane to the tmp window above, and make the fake the appear
# in its place
if [[ "$TZOOM" = "1" ]] ; then
    # the -Z flag to swap-pane only appeared with tmux 3.1
    if ! tmux swap-pane -t "${TMUX_WORMHOLE_ORIG_WINDOW}" -Z 2> /dev/null ; then
	tmux swap-pane -t "${TMUX_WORMHOLE_ORIG_WINDOW}"
	tmux resize-pane -Z
    fi
else
    tmux swap-pane -t "${TMUX_WORMHOLE_ORIG_WINDOW}"
fi

exit 0
