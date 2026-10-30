/*
    Player state persistence.

    Saves the current playback state (playing track, playback position,
    volume, repeat state) into localStorage while the page is alive.
    After a refresh the state is restored automatically, but the playback
    stays paused - the user only has to press play to continue where
    they left off.
*/

var playerState = null;

// While a saved state is being restored (media is loading and the position
// is being seeked) capturing is suspended, otherwise the periodic saves
// would overwrite the saved state with time: 0.
var playerStateRestoring = false;

const PLAYER_STATE_KEY = "euterpe_player_state";
const PLAYER_STATE_INTERVAL_MS = 1000;

// playerStateSave stores the given state in the localStorage.
function playerStateSave(state) {
    if (!localStorage) {
        return;
    }
    try {
        localStorage[PLAYER_STATE_KEY] = JSON.stringify(state);
    } catch (err) {
        // Ignore storage errors (full quota, private browsing, ...)
    }
}

// playerStateLoad reads and parses the saved state or returns null.
function playerStateLoad() {
    if (!localStorage) {
        return null;
    }
    try {
        var raw = localStorage[PLAYER_STATE_KEY];
        if (!raw) {
            return null;
        }
        var state = JSON.parse(raw);
        return state || null;
    } catch (err) {
        return null;
    }
}

// playerStateStart begins tracking the player and starts the
// periodic save.
function playerStateStart(playlistObj) {
    if (playerState) {
        return;
    }

    var jPlayerSel = playlistObj.cssSelector.jPlayer;

    var lastSaved = 0;

    // jPlayer("status") does not work (status is a property, not a method,
    // of the jPlayer instance). The instance is stored as jQuery data.
    var getJPlayerInstance = function () {
        return $(jPlayerSel).data("jPlayer") || null;
    };

    var captureState = function (force) {
        var inst = getJPlayerInstance();
        if (!inst) {
            return;
        }

        var now = Date.now();
        if (!force && now - lastSaved < PLAYER_STATE_INTERVAL_MS) {
            return;
        }
        lastSaved = now;

        if (playerStateRestoring) {
            return;
        }

        var status = inst.status;

        var media = status.media;
        if (!media || media.media_id === undefined || media.media_id === null) {
            // No track is loaded (page just started). Do not overwrite the
            // saved state.
            return;
        }

        var state = {
            tr: media.media_id,
            time: status.currentTime || 0,
            volume: null
        };

        // The volume and the loop flag are stored in the options,
        // not in the status.
        if (inst.options) {
            if (typeof inst.options.volume === "number") {
                state.volume = inst.options.volume;
            }
            state.repeat = inst.options.loop ? "true" : "false";
        }

        if (playerState) {
            playerStateSave(state);
        }
    };

    var started = {
        stop: null
    };
    playerState = started;

    $(jPlayerSel).bind($.jPlayer.event.timeupdate, function () {
        captureState(false);
    });

    $(jPlayerSel).bind($.jPlayer.event.volumechange, function () {
        captureState(false);
    });

    $(jPlayerSel).bind($.jPlayer.event.ended, function () {
        // The track is done, no point in resuming it next time.
        playerStateClear();
    });

    $(jPlayerSel).bind($.jPlayer.event.stop, function () {
        playerStateClear();
    });

    // Make sure the state is not lost when the tab is closed or the page
    // is navigated away from.
    $(window).on("beforeunload pagehide", function () {
        captureState(true);
    });

    var interval = setInterval(function () {
        captureState(false);
    }, PLAYER_STATE_INTERVAL_MS);

    started.stop = function () {
        clearInterval(interval);
        if (playerState === started) {
            playerState = null;
        }
    };
}

// playerStateClear removes the saved state (e.g. when playback stopped).
function playerStateClear() {
    if (localStorage) {
        delete localStorage[PLAYER_STATE_KEY];
    }
}

// playerStateRestore tries to restore a previously saved state. It only
// selects the track and seeks to the saved position - the playback stays
// paused.
function playerStateRestore(playlistObj) {
    var state = playerStateLoad();
    if (!state || state.tr === null || state.tr === undefined) {
        return false;
    }

    var index = -1;
    for (var i = 0; i < playlistObj.playlist.length; i++) {
        if (playlistObj.playlist[i].media_id == state.tr) {
            index = i;
            break;
        }
    }

    if (index < 0) {
        playerStateClear();
        return false;
    }

    playlistObj._highlight(index);
    playlistObj.current = index;

    // Suspend the periodic saves while the media is being loaded and
    // seeked, otherwise they would clobber the saved position.
    playerStateRestoring = true;

    var jPlayerSel = playlistObj.cssSelector.jPlayer;
    var jPlayer = $(jPlayerSel);

    jPlayer.jPlayer("setMedia", playlistObj.playlist[index]);

    if (typeof state.volume === "number") {
        jPlayer.jPlayer("volume", state.volume);
    }

    // preload is "none" so the browser would not fetch the metadata on
    // its own (and jPlayer's load command does not work reliably here).
    // Force a fetch directly on the audio element so the duration
    // becomes known and seeking is possible.
    var getMediaEl = function () {
        var inst = $(jPlayerSel).data("jPlayer");
        return (inst && inst.htmlElement && inst.htmlElement.media) || null;
    };

    // playHead takes a percentage (0-100). The media needs its duration
    // first, so poll until it is known.
    if (state.time > 0) {
        var attempts = 0;
        var waitSeek = function () {
            var inst = $(jPlayerSel).data("jPlayer");
            var mediaEl = getMediaEl();
            // If the user switched to another track meanwhile, give up.
            if (inst && (!inst.status.media || inst.status.media.media_id != state.tr)) {
                playerStateRestoring = false;
                return;
            }
            if (inst && inst.status.duration > 0) {
                jPlayer.jPlayer("playHead", state.time * 100 / inst.status.duration);
                // Once the seek is issued the position is valid again,
                // re-enable the periodic saves.
                setTimeout(function () {
                    playerStateRestoring = false;
                }, 1000);
                return;
            }
            // The media element is not loading (another setMedia aborted
            // the in-flight fetch). Force a new one.
            if (mediaEl && mediaEl.networkState === 1 && attempts % 5 === 0) {
                try {
                    mediaEl.load();
                } catch (err) {
                    // Ignore.
                }
            }
            attempts++;
            if (attempts < 100) {
                setTimeout(waitSeek, 100);
            } else {
                playerStateRestoring = false;
            }
        };
        waitSeek();
    } else {
        playerStateRestoring = false;
    }

    return true;
}
