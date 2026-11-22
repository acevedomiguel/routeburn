// Progressive enhancement: copy buttons for command blocks. The site is fully
// usable without this script — commands remain selectable text.
(function () {
  'use strict';

  var live = document.createElement('p');
  live.className = 'visually-hidden';
  live.setAttribute('role', 'status');
  document.body.appendChild(live);

  document.querySelectorAll('.command').forEach(function (block) {
    var pre = block.querySelector('pre');
    if (!pre || !navigator.clipboard || !navigator.clipboard.writeText) {
      return;
    }

    var button = document.createElement('button');
    button.type = 'button';
    button.className = 'copy-button';
    button.textContent = 'copy';
    block.appendChild(button);

    button.addEventListener('click', function () {
      navigator.clipboard.writeText(pre.textContent).then(function () {
        button.textContent = 'copied';
        live.textContent = 'Command copied to clipboard.';
        window.setTimeout(function () {
          button.textContent = 'copy';
        }, 2000);
      }, function () {
        live.textContent = 'Copy failed; select the command text manually.';
      });
    });
  });
})();
