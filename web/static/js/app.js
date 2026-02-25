// SpeakEasy - Language Learning App JavaScript

// Audio playback for TTS
function playAudio(e, text, lang) {
    e.stopPropagation();
    var btn = e.currentTarget;
    btn.classList.add('playing');

    var audio = new Audio('/api/tts?text=' + encodeURIComponent(text) + '&lang=' + encodeURIComponent(lang));
    audio.addEventListener('ended', function() {
        btn.classList.remove('playing');
    });
    audio.addEventListener('error', function() {
        btn.classList.remove('playing');
    });
    audio.play().catch(function() {
        btn.classList.remove('playing');
    });
}

// Script toggle (Cyrillic / Latin / Both)
function setScript(e, mode) {
    document.body.className = document.body.className
        .replace(/show-(cyrillic|latin|both)/g, '')
        .trim();
    document.body.classList.add('show-' + mode);

    document.querySelectorAll('.script-toggle button').forEach(function(btn) {
        btn.classList.remove('active');
    });
    e.currentTarget.classList.add('active');

    fetch('/api/preference/script', {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: 'mode=' + mode
    });
}

// Quiz: select multiple choice option
function selectOption(questionIdx, optionIdx) {
    var container = document.getElementById('question-' + questionIdx);
    if (!container) return;

    container.querySelectorAll('.quiz-option').forEach(function(opt) {
        opt.classList.remove('selected');
    });

    var selected = container.querySelector('[data-option="' + optionIdx + '"]');
    if (selected) {
        selected.classList.add('selected');
    }

    var input = document.getElementById('answer-' + questionIdx);
    if (input) {
        input.value = optionIdx;
    }
}

// Quiz: drag-and-drop match pairs
(function() {
    var dragState = {}; // { qIdx: { matched: {englishIdx: targetIdx}, selectedChip: null } }

    function getState(qIdx) {
        if (!dragState[qIdx]) dragState[qIdx] = { matched: {}, selectedChip: null };
        return dragState[qIdx];
    }

    function updateAnswer(qIdx) {
        var state = getState(qIdx);
        var matched = [];
        for (var ei in state.matched) {
            matched.push({ english: parseInt(ei), target: state.matched[ei] });
        }
        var input = document.getElementById('answer-' + qIdx);
        if (input) input.value = matched.length ? JSON.stringify(matched) : '';
    }

    function clearZone(zone) {
        zone.innerHTML = '<span class="drop-hint">Drop here</span>';
        zone.classList.remove('filled', 'drag-over');
    }

    function returnChipToBank(qIdx, targetIdx) {
        var chip = document.querySelector('.drag-chip[data-q="' + qIdx + '"][data-ti="' + targetIdx + '"]');
        var bank = document.getElementById('match-bank-' + qIdx);
        if (chip && bank) {
            chip.style.display = '';
            bank.appendChild(chip);
        }
    }

    function placeChip(qIdx, englishIdx, targetIdx) {
        var state = getState(qIdx);

        // Return existing chip in this slot back to bank
        if (state.matched[englishIdx] !== undefined) {
            returnChipToBank(qIdx, state.matched[englishIdx]);
            var oldZone = document.querySelector('.drag-drop-zone[data-q="' + qIdx + '"][data-english="' + englishIdx + '"]');
            if (oldZone) clearZone(oldZone);
        }

        // If this target chip is already placed elsewhere, clear that slot first
        for (var ei in state.matched) {
            if (state.matched[ei] === targetIdx && parseInt(ei) !== englishIdx) {
                delete state.matched[ei];
                var otherZone = document.querySelector('.drag-drop-zone[data-q="' + qIdx + '"][data-english="' + ei + '"]');
                if (otherZone) clearZone(otherZone);
                break;
            }
        }

        state.matched[englishIdx] = targetIdx;

        var zone = document.querySelector('.drag-drop-zone[data-q="' + qIdx + '"][data-english="' + englishIdx + '"]');
        var chip = document.querySelector('.drag-chip[data-q="' + qIdx + '"][data-ti="' + targetIdx + '"]');
        if (zone && chip) {
            chip.style.display = 'none';
            zone.innerHTML = '<div class="placed-chip"><span>' + chip.dataset.label + '</span>'
                + '<button type="button" class="remove-chip-btn" onclick="removeChip(' + qIdx + ',' + englishIdx + ')" title="Remove">&#x2715;</button></div>';
            zone.classList.add('filled');
            zone.classList.remove('drag-over');
        }

        updateAnswer(qIdx);
    }

    // HTML5 drag-and-drop
    window.handleDragStart = function(event) {
        var el = event.currentTarget;
        event.dataTransfer.setData('text/plain', JSON.stringify({
            qIdx: parseInt(el.dataset.q),
            targetIdx: parseInt(el.dataset.ti)
        }));
        event.dataTransfer.effectAllowed = 'move';
    };

    window.allowDrop = function(event) {
        event.preventDefault();
        event.currentTarget.classList.add('drag-over');
    };

    window.handleDragLeave = function(event) {
        event.currentTarget.classList.remove('drag-over');
    };

    window.handleDrop = function(event) {
        event.preventDefault();
        var zone = event.currentTarget;
        zone.classList.remove('drag-over');
        try {
            var data = JSON.parse(event.dataTransfer.getData('text/plain'));
            placeChip(data.qIdx, parseInt(zone.dataset.english), data.targetIdx);
        } catch(e) {}
    };

    // Click-to-place (tap on mobile, accessibility fallback)
    window.clickChip = function(qIdx, targetIdx) {
        var state = getState(qIdx);
        var chip = document.querySelector('.drag-chip[data-q="' + qIdx + '"][data-ti="' + targetIdx + '"]');
        if (!chip) return;

        if (state.selectedChip === targetIdx) {
            chip.classList.remove('chip-selected');
            state.selectedChip = null;
        } else {
            if (state.selectedChip !== null) {
                var prev = document.querySelector('.drag-chip[data-q="' + qIdx + '"][data-ti="' + state.selectedChip + '"]');
                if (prev) prev.classList.remove('chip-selected');
            }
            state.selectedChip = targetIdx;
            chip.classList.add('chip-selected');
        }
    };

    window.clickDropZone = function(qIdx, englishIdx) {
        var state = getState(qIdx);
        if (state.selectedChip !== null) {
            var chip = document.querySelector('.drag-chip[data-q="' + qIdx + '"][data-ti="' + state.selectedChip + '"]');
            if (chip) chip.classList.remove('chip-selected');
            placeChip(qIdx, englishIdx, state.selectedChip);
            state.selectedChip = null;
        }
    };

    window.removeChip = function(qIdx, englishIdx) {
        var state = getState(qIdx);
        var targetIdx = state.matched[englishIdx];
        if (targetIdx === undefined) return;
        returnChipToBank(qIdx, targetIdx);
        delete state.matched[englishIdx];
        var zone = document.querySelector('.drag-drop-zone[data-q="' + qIdx + '"][data-english="' + englishIdx + '"]');
        if (zone) clearZone(zone);
        updateAnswer(qIdx);
    };
})();

// Confetti effect
function showConfetti() {
    var colors = ['#7C3AED', '#F59E0B', '#14B8A6', '#3B82F6', '#EF4444', '#10B981'];
    for (var i = 0; i < 50; i++) {
        var piece = document.createElement('div');
        piece.className = 'confetti-piece';
        piece.style.left = Math.random() * 100 + 'vw';
        piece.style.backgroundColor = colors[Math.floor(Math.random() * colors.length)];
        piece.style.animationDelay = Math.random() * 2 + 's';
        piece.style.borderRadius = Math.random() > 0.5 ? '50%' : '0';
        piece.style.width = (Math.random() * 8 + 6) + 'px';
        piece.style.height = (Math.random() * 8 + 6) + 'px';
        document.body.appendChild(piece);

        setTimeout(function(el) {
            el.remove();
        }, 5000, piece);
    }
}

// Vocab card flip
document.addEventListener('click', function(e) {
    var card = e.target.closest('.vocab-card');
    if (card && !e.target.closest('.play-btn')) {
        card.classList.toggle('flipped');
    }
});

// Quiz form validation — warn before submitting unanswered questions
document.addEventListener('DOMContentLoaded', function() {
    var form = document.getElementById('quiz-form');
    if (form) {
        form.addEventListener('submit', function(event) {
            var unanswered = 0;
            var cards = form.querySelectorAll('.question-card');
            cards.forEach(function(card) {
                var type = card.querySelector('input[name^="type-"]');
                if (!type) return;
                var qType = type.value;
                var idx = type.name.replace('type-', '');
                var answer = form.querySelector('input[name="answer-' + idx + '"]');
                if (!answer) return;

                if (qType === 'multiple_choice' || qType === 'listen_and_choose') {
                    if (answer.value === '') unanswered++;
                } else if (qType === 'match_pairs') {
                    // Check if all pairs are matched by counting entries in the JSON
                    var val = answer.value;
                    if (!val) { unanswered++; return; }
                    try {
                        var matched = JSON.parse(val);
                        var pairsCount = card.querySelectorAll('.drag-pair-row').length;
                        if (matched.length < pairsCount) unanswered++;
                    } catch(e) { unanswered++; }
                }
            });

            if (unanswered > 0) {
                var msg = unanswered === 1
                    ? '1 question is unanswered. Submit anyway?'
                    : unanswered + ' questions are unanswered. Submit anyway?';
                if (!confirm(msg)) {
                    event.preventDefault();
                }
            }
        });
    }

    // Auto-trigger confetti on results page if passed
    if (document.querySelector('.results-score.pass')) {
        setTimeout(showConfetti, 500);
    }
});
