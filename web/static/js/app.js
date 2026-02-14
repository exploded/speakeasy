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

// Quiz: match pairs
(function() {
    var matchState = {};

    window.selectMatchItem = function(questionIdx, side, index) {
        var key = 'q' + questionIdx;
        if (!matchState[key]) {
            matchState[key] = { matched: [] };
        }
        var state = matchState[key];

        var el = document.querySelector('[data-match="' + questionIdx + '-' + side + '-' + index + '"]');
        if (!el || el.classList.contains('matched')) return;

        if (state.selected && state.selected.side === side) {
            document.querySelector('[data-match="' + questionIdx + '-' + state.selected.side + '-' + state.selected.index + '"]')
                .classList.remove('selected');
            state.selected = null;
            return;
        }

        if (!state.selected) {
            el.classList.add('selected');
            state.selected = { side: side, index: index };
        } else {
            var prevEl = document.querySelector('[data-match="' + questionIdx + '-' + state.selected.side + '-' + state.selected.index + '"]');

            var englishIdx = side === 'english' ? index : state.selected.index;
            var targetIdx = side === 'target' ? index : state.selected.index;

            state.matched.push({ english: englishIdx, target: targetIdx });

            var input = document.getElementById('answer-' + questionIdx);
            if (input) {
                input.value = JSON.stringify(state.matched);
            }

            prevEl.classList.remove('selected');
            prevEl.classList.add('matched');
            el.classList.remove('selected');
            el.classList.add('matched');

            state.selected = null;
        }
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

// Auto-trigger confetti on results page if passed
document.addEventListener('DOMContentLoaded', function() {
    if (document.querySelector('.results-score.pass')) {
        setTimeout(showConfetti, 500);
    }
});
