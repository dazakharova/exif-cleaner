(function () {
    const input = document.getElementById('file');
    const chosen = document.getElementById('chosen');

    input.addEventListener('change', () => {
        const f = input.files && input.files[0];
        if (!f) {
            chosen.hidden = true;
            chosen.textContent = '';
            return;
        }

        const sizeKB = (f.size / 1024).toFixed(1);
        chosen.textContent = `${f.name} • ${sizeKB} KB`;
        chosen.hidden = false;
    });
})();
