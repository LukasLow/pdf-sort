let currentFile = "";
let globalConfig = {};

async function loadNextFile() {
    const response = await fetch('/next-pdf');
    const data = await response.json();
    if (!data.filename) return;
    currentFile = data.filename;
    document.getElementById('pdf-viewer').src = "/view-pdf/" + encodeURIComponent(currentFile);
    document.getElementById('date-year').value = data.year;
    document.getElementById('date-month').value = data.month;
    document.getElementById('date-day').value = data.day;
    document.getElementById('file-size').innerText = data.file_size || "-";
    document.getElementById('ppi').innerText = data.ppi || "-";
    globalConfig = data.config_data || {};
    renderDropdowns(data.suggested_correspondent, data.suggested_info);
    updatePreview();
}

function renderDropdowns(selectedCorr, selectedInfo) {
    const corrSel = document.getElementById('sel-correspondent');
    corrSel.innerHTML = "";
    Object.keys(globalConfig).sort().forEach(corr => {
        let opt = document.createElement('option');
        opt.value = corr;
        opt.innerText = corr;
        corrSel.appendChild(opt);
    });
    if (selectedCorr) corrSel.value = selectedCorr;
    onCorrespondentChange();
    if (selectedInfo) document.getElementById('sel-info').value = selectedInfo;
}

function onCorrespondentChange() {
    const corr = document.getElementById('sel-correspondent').value;
    const infoSelect = document.getElementById('sel-info');
    infoSelect.innerHTML = "";
    if (globalConfig[corr]) {
        globalConfig[corr].forEach(info => {
            let opt = document.createElement('option');
            opt.value = info;
            opt.innerText = info;
            infoSelect.appendChild(opt);
        });
    }
    updatePreview();
}

function updatePreview() {
    const year = document.getElementById('date-year').value;
    const month = document.getElementById('date-month').value;
    const day = document.getElementById('date-day').value;
    const corr = document.getElementById('sel-correspondent').value;
    const info = document.getElementById('sel-info').value;
    const extra = document.getElementById('extra-text').value;
    let txt = `${year}-${month}-${day}_${corr}_${info}`;
    if (extra) txt += `_${extra}`;
    txt += ".pdf";
    document.getElementById('filename-preview').innerText = txt;
}

async function processFile() {
    const payload = {
        filename: currentFile,
        year: document.getElementById('date-year').value,
        month: document.getElementById('date-month').value,
        day: document.getElementById('date-day').value,
        correspondent: document.getElementById('sel-correspondent').value,
        info: document.getElementById('sel-info').value,
        extra: document.getElementById('extra-text').value,
        compress: document.getElementById('compress').checked
    };
    await fetch('/process', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(payload)});
    loadNextFile();
}

loadNextFile();

async function loadBuildInfo() {
    try {
        const resp = await fetch('/buildinfo');
        if (!resp.ok) return;
        const data = await resp.json();
        const el = document.getElementById('build-info');
        if (el) el.textContent = `Build: ${data.build}`;
    } catch (e) {}
}
loadBuildInfo();

function deleteFile() {
    if (!currentFile) return;
    if (!window.confirm('Bist du sicher, dass du diese Datei löschen möchtest?')) return;
    fetch(`/trash?filename=${encodeURIComponent(currentFile)}`).then(() => loadNextFile()).catch(() => {});
}
