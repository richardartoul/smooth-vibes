// State
let currentStatus = null;
let selectedFiles = new Set();
let pendingConfirm = null;
let settingsDirty = false;
let originalConfig = null;

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    refreshStatus();
    loadInitialConfig();
    setInterval(refreshStatus, 5000); // Poll every 5 seconds
});

// Load config on startup to set initial UI state
async function loadInitialConfig() {
    try {
        const cfg = await api('/config');
        updateExperimentsVisibility(cfg.experimentsEnabled);
    } catch (e) {
        console.error('Failed to load initial config:', e);
    }
}

// API helpers
async function api(endpoint, options = {}) {
    const response = await fetch(`/api${endpoint}`, {
        headers: { 'Content-Type': 'application/json' },
        ...options
    });
    const data = await response.json();
    if (!response.ok) {
        throw new Error(data.error || 'Request failed');
    }
    return data;
}

// Panel navigation
function showPanel(panelId, skipDirtyCheck = false) {
    // Check for unsaved settings when leaving settings panel
    const settingsPanel = document.getElementById('settingsPanel');
    if (!skipDirtyCheck && settingsPanel && !settingsPanel.classList.contains('hidden') && panelId !== 'settingsPanel') {
        checkSettingsDirty();
        if (settingsDirty) {
            showConfirm(
                'Unsaved changes',
                'You have unsaved settings changes. Do you want to discard them?',
                () => {
                    settingsDirty = false;
                    originalConfig = null;
                    showPanel(panelId, true);
                }
            );
            return;
        }
    }
    
    document.querySelectorAll('.panel').forEach(p => p.classList.add('hidden'));
    document.getElementById(panelId).classList.remove('hidden');
    
    // Load data for specific panels
    if (panelId === 'savePanel') loadChanges();
    if (panelId === 'restorePanel') loadCommits();
    if (panelId === 'backupsPanel') loadBackups();
    if (panelId === 'experimentsPanel') loadExperiments();
    if (panelId === 'settingsPanel') loadConfig();
}

// Status
async function refreshStatus() {
    try {
        currentStatus = await api('/status');
        updateStatusUI();
    } catch (e) {
        console.error('Failed to refresh status:', e);
    }
}

function updateStatusUI() {
    const branchBadge = document.getElementById('branchBadge');
    const changesBadge = document.getElementById('changesBadge');
    const keepBtn = document.getElementById('keepBtn');
    const abandonBtn = document.getElementById('abandonBtn');
    
    branchBadge.textContent = currentStatus.branch;
    branchBadge.classList.toggle('experiment', !currentStatus.isOnMain);
    
    changesBadge.classList.toggle('hidden', !currentStatus.hasChanges);
    
    // Show/hide experiment buttons
    keepBtn.classList.toggle('hidden', currentStatus.isOnMain);
    abandonBtn.classList.toggle('hidden', currentStatus.isOnMain);
}

// Save Progress
async function loadChanges() {
    const fileList = document.getElementById('fileList');
    fileList.innerHTML = '<p class="loading">Loading changes...</p>';
    
    try {
        const changes = await api('/changes');
        selectedFiles = new Set(changes.map(c => c.Path));
        
        if (changes.length === 0) {
            fileList.innerHTML = '<div class="empty-state"><p>No changes to save!</p><p>Your work is already saved.</p></div>';
            return;
        }
        
        fileList.innerHTML = changes.map(change => {
            const statusIcon = change.Status === 'added' ? '+' : change.Status === 'deleted' ? '-' : '~';
            const statusClass = change.Status;
            return `
                <div class="file-item selected" data-path="${change.Path}" onclick="toggleFile(this)">
                    <input type="checkbox" checked>
                    <span class="file-status ${statusClass}">${statusIcon}</span>
                    <span class="file-path">${change.Path}</span>
                </div>
            `;
        }).join('');
    } catch (e) {
        fileList.innerHTML = `<div class="empty-state"><p>Error loading changes</p><p>${e.message}</p></div>`;
    }
}

function toggleFile(element) {
    const path = element.dataset.path;
    const checkbox = element.querySelector('input[type="checkbox"]');
    
    if (selectedFiles.has(path)) {
        selectedFiles.delete(path);
        checkbox.checked = false;
        element.classList.remove('selected');
        
        // Ask about gitignore
        showConfirm(
            'Add to .gitignore?',
            `Would you like to add "${path}" to .gitignore so it's never tracked?`,
            async () => {
                try {
                    await api('/gitignore', {
                        method: 'POST',
                        body: JSON.stringify({ pattern: path })
                    });
                    showToast('Added to .gitignore', 'success');
                } catch (e) {
                    showToast(e.message, 'error');
                }
            }
        );
    } else {
        selectedFiles.add(path);
        checkbox.checked = true;
        element.classList.add('selected');
    }
}

async function saveProgress() {
    const message = document.getElementById('commitMessage').value.trim();
    if (!message) {
        showToast('Please enter a description', 'error');
        return;
    }
    
    if (selectedFiles.size === 0) {
        showToast('No files selected', 'error');
        return;
    }
    
    showLoading(true);
    try {
        const result = await api('/save', {
            method: 'POST',
            body: JSON.stringify({
                message,
                files: Array.from(selectedFiles)
            })
        });
        
        // Show appropriate message based on auto-sync result
        if (result.autoSynced) {
            if (result.syncError) {
                showToast('Saved! (Auto-sync failed: ' + result.syncError + ')', 'success');
            } else {
                showToast('Saved and synced to GitHub!', 'success');
            }
        } else {
            showToast('Progress saved!', 'success');
        }
        
        document.getElementById('commitMessage').value = '';
        refreshStatus();
        showPanel('menuPanel');
    } catch (e) {
        showToast(e.message, 'error');
    }
    showLoading(false);
}

// Restore
async function loadCommits() {
    const commitList = document.getElementById('commitList');
    commitList.innerHTML = '<p class="loading">Loading save points...</p>';
    
    try {
        const commits = await api('/commits');
        
        if (commits.length === 0) {
            commitList.innerHTML = '<div class="empty-state"><p>No save points found</p><p>Save your progress first!</p></div>';
            return;
        }
        
        commitList.innerHTML = commits.map(commit => `
            <div class="commit-item" onclick="restoreCommit('${commit.FullHash}', '${escapeHtml(commit.Message)}')">
                <div class="commit-info">
                    <span class="commit-hash">${commit.Hash}</span>
                    <span class="commit-message">${escapeHtml(commit.Message)}</span>
                    <span class="commit-time">${commit.Timestamp}</span>
                </div>
                <button class="restore-btn">Restore</button>
            </div>
        `).join('');
    } catch (e) {
        commitList.innerHTML = `<div class="empty-state"><p>Error loading commits</p><p>${e.message}</p></div>`;
    }
}

function restoreCommit(hash, message) {
    showConfirm(
        'Restore to this state?',
        `This will restore to "${message}". A backup will be created first. Current unsaved changes will be lost.`,
        async () => {
            showLoading(true);
            try {
                const result = await api('/restore', {
                    method: 'POST',
                    body: JSON.stringify({ commitHash: hash })
                });
                showToast(`Restored! Backup: ${result.backup}`, 'success');
                refreshStatus();
                showPanel('menuPanel');
            } catch (e) {
                showToast(e.message, 'error');
            }
            showLoading(false);
        }
    );
}

// Backups
async function loadBackups() {
    const backupList = document.getElementById('backupList');
    backupList.innerHTML = '<p class="loading">Loading backups...</p>';
    
    try {
        const backups = await api('/backups');
        
        if (backups.length === 0) {
            backupList.innerHTML = '<div class="empty-state"><p>No backups yet</p><p>Backups are created when you restore to a previous state.</p></div>';
            return;
        }
        
        backupList.innerHTML = backups.map(backup => `
            <div class="backup-item" onclick="restoreBackup('${backup.Name}', '${formatTimestamp(backup.Timestamp)}')">
                <div class="backup-info">
                    <span class="backup-time">${formatTimestamp(backup.Timestamp)}</span>
                    <span class="backup-message">${escapeHtml(backup.Message)}</span>
                    <span class="backup-branch">${backup.CommitHash}</span>
                </div>
                <button class="restore-btn">Restore</button>
            </div>
        `).join('');
    } catch (e) {
        backupList.innerHTML = `<div class="empty-state"><p>Error loading backups</p><p>${e.message}</p></div>`;
    }
}

function restoreBackup(name, timestamp) {
    showConfirm(
        'Restore from backup?',
        `This will restore from backup created at ${timestamp}. Current unsaved changes will be lost.`,
        async () => {
            showLoading(true);
            try {
                await api('/restore-backup', {
                    method: 'POST',
                    body: JSON.stringify({ backupName: name })
                });
                showToast('Restored from backup!', 'success');
                refreshStatus();
                showPanel('menuPanel');
            } catch (e) {
                showToast(e.message, 'error');
            }
            showLoading(false);
        }
    );
}

// Experiments
async function loadExperiments() {
    const experimentList = document.getElementById('experimentList');
    experimentList.innerHTML = '<p class="loading">Loading experiments...</p>';
    
    try {
        const experiments = await api('/experiments');
        
        if (experiments.length === 0) {
            experimentList.innerHTML = '<div class="empty-state"><p>No experiments yet</p><p>Start a new experiment above!</p></div>';
            return;
        }
        
        experimentList.innerHTML = experiments.map(exp => `
            <div class="experiment-item ${exp.IsCurrent ? 'current' : ''}" onclick="switchExperiment('${exp.Name}')">
                <div class="experiment-info">
                    <span class="experiment-name">${exp.Name}</span>
                </div>
                ${exp.IsCurrent ? '<span class="current-badge">current</span>' : '<button class="restore-btn">Switch</button>'}
            </div>
        `).join('');
    } catch (e) {
        experimentList.innerHTML = `<div class="empty-state"><p>Error loading experiments</p><p>${e.message}</p></div>`;
    }
}

async function createExperiment() {
    const name = document.getElementById('experimentName').value.trim();
    if (!name) {
        showToast('Please enter an experiment name', 'error');
        return;
    }
    
    showLoading(true);
    try {
        const result = await api('/experiment/create', {
            method: 'POST',
            body: JSON.stringify({ name })
        });
        showToast(`Created: ${result.branch}`, 'success');
        document.getElementById('experimentName').value = '';
        refreshStatus();
        loadExperiments();
    } catch (e) {
        showToast(e.message, 'error');
    }
    showLoading(false);
}

async function keepExperiment() {
    showConfirm(
        'Keep this experiment?',
        'This will merge your experiment into main. Make sure you have saved your progress first.',
        async () => {
            showLoading(true);
            try {
                await api('/experiment/keep', { method: 'POST' });
                showToast('Experiment merged into main!', 'success');
                refreshStatus();
                showPanel('menuPanel');
            } catch (e) {
                showToast(e.message, 'error');
            }
            showLoading(false);
        }
    );
}

async function abandonExperiment() {
    showConfirm(
        'Abandon this experiment?',
        'This will delete your experiment and switch back to main. This cannot be undone!',
        async () => {
            showLoading(true);
            try {
                await api('/experiment/abandon', { method: 'POST' });
                showToast('Experiment abandoned', 'success');
                refreshStatus();
                showPanel('menuPanel');
            } catch (e) {
                showToast(e.message, 'error');
            }
            showLoading(false);
        }
    );
}

async function switchExperiment(branch) {
    showLoading(true);
    try {
        await api('/experiment/switch', {
            method: 'POST',
            body: JSON.stringify({ branch })
        });
        showToast(`Switched to ${branch}`, 'success');
        refreshStatus();
        loadExperiments();
    } catch (e) {
        showToast(e.message, 'error');
    }
    showLoading(false);
}

// Settings
async function loadConfig() {
    try {
        const cfg = await api('/config');
        document.getElementById('autoSyncToggle').checked = cfg.autoSyncEnabled;
        document.getElementById('maxBackupsInput').value = cfg.maxBackups;
        document.getElementById('experimentsToggle').checked = cfg.experimentsEnabled;
        // Store original values to detect changes
        originalConfig = { ...cfg };
        settingsDirty = false;
        // Update UI visibility based on experiments setting
        updateExperimentsVisibility(cfg.experimentsEnabled);
    } catch (e) {
        console.error('Failed to load config:', e);
        showToast('Failed to load settings', 'error');
    }
}

function updateExperimentsVisibility(enabled) {
    const experimentsBtn = document.querySelector('.menu-btn[onclick="showPanel(\'experimentsPanel\')"]');
    if (experimentsBtn) {
        experimentsBtn.classList.toggle('hidden', !enabled);
    }
}

function checkSettingsDirty() {
    if (!originalConfig) return false;
    const autoSyncEnabled = document.getElementById('autoSyncToggle').checked;
    const maxBackups = parseInt(document.getElementById('maxBackupsInput').value) || 10;
    const experimentsEnabled = document.getElementById('experimentsToggle').checked;
    settingsDirty = (autoSyncEnabled !== originalConfig.autoSyncEnabled) || 
                    (maxBackups !== originalConfig.maxBackups) ||
                    (experimentsEnabled !== originalConfig.experimentsEnabled);
    return settingsDirty;
}

async function updateConfig() {
    const autoSyncEnabled = document.getElementById('autoSyncToggle').checked;
    const maxBackups = parseInt(document.getElementById('maxBackupsInput').value) || 10;
    const experimentsEnabled = document.getElementById('experimentsToggle').checked;
    
    try {
        const cfg = await api('/config', {
            method: 'POST',
            body: JSON.stringify({ autoSyncEnabled, maxBackups, experimentsEnabled })
        });
        // Update original config after successful save
        originalConfig = { ...cfg };
        settingsDirty = false;
        // Update UI visibility based on experiments setting
        updateExperimentsVisibility(cfg.experimentsEnabled);
        showToast('Settings saved', 'success');
    } catch (e) {
        showToast('Failed to save settings: ' + e.message, 'error');
    }
}

function tryLeaveSettings(targetPanel) {
    checkSettingsDirty();
    if (settingsDirty) {
        showConfirm(
            'Unsaved changes',
            'You have unsaved settings changes. Do you want to discard them?',
            () => {
                settingsDirty = false;
                originalConfig = null;
                showPanel(targetPanel);
            }
        );
    } else {
        showPanel(targetPanel);
    }
}

// Sync
async function syncToGitHub(remoteUrl = null) {
    showLoading(true);
    try {
        const body = remoteUrl ? { remoteUrl } : {};
        const result = await api('/sync', { 
            method: 'POST',
            body: JSON.stringify(body)
        });
        
        // Check if we need to set up a remote
        if (result.needsRemote) {
            showLoading(false);
            showPrompt(
                'Set up GitHub',
                'Enter your GitHub repository SSH URL (click Code → SSH on GitHub):',
                'git@github.com:username/repo.git',
                async (url) => {
                    if (url) {
                        await syncToGitHub(url);
                    }
                }
            );
            return;
        }
        
        showToast('Synced to GitHub!', 'success');
    } catch (e) {
        showToast(e.message, 'error');
    }
    showLoading(false);
}

// UI Helpers
function showToast(message, type = '') {
    const toast = document.getElementById('toast');
    toast.textContent = message;
    toast.className = `toast ${type} show`;
    setTimeout(() => toast.classList.remove('show'), 3000);
}

function showLoading(show) {
    document.getElementById('loadingOverlay').classList.toggle('hidden', !show);
}

function showConfirm(title, message, onConfirm) {
    const modal = document.getElementById('confirmModal');
    document.getElementById('confirmTitle').textContent = title;
    document.getElementById('confirmMessage').textContent = message;
    document.getElementById('confirmInput').classList.add('hidden');
    document.getElementById('confirmBtn').textContent = 'Confirm';
    document.getElementById('confirmBtn').className = 'btn-danger';
    
    const confirmBtn = document.getElementById('confirmBtn');
    confirmBtn.onclick = () => {
        closeModal();
        onConfirm();
    };
    
    modal.classList.remove('hidden');
}

function showPrompt(title, message, placeholder, onSubmit) {
    const modal = document.getElementById('confirmModal');
    document.getElementById('confirmTitle').textContent = title;
    document.getElementById('confirmMessage').textContent = message;
    
    const input = document.getElementById('confirmInput');
    input.classList.remove('hidden');
    input.placeholder = placeholder;
    input.value = '';
    input.focus();
    
    document.getElementById('confirmBtn').textContent = 'Save & Sync';
    document.getElementById('confirmBtn').className = 'action-btn';
    
    const confirmBtn = document.getElementById('confirmBtn');
    confirmBtn.onclick = () => {
        const value = input.value.trim();
        closeModal();
        onSubmit(value);
    };
    
    modal.classList.remove('hidden');
    setTimeout(() => input.focus(), 100);
}

function closeModal() {
    document.getElementById('confirmModal').classList.add('hidden');
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function formatTimestamp(ts) {
    // Input: 20060102-150405
    if (ts.length >= 15) {
        const date = ts.slice(0, 8);
        const time = ts.slice(9, 15);
        return `${date.slice(0,4)}-${date.slice(4,6)}-${date.slice(6,8)} ${time.slice(0,2)}:${time.slice(2,4)}:${time.slice(4,6)}`;
    }
    return ts;
}

// Keyboard shortcuts
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
        const modal = document.getElementById('confirmModal');
        if (!modal.classList.contains('hidden')) {
            closeModal();
            return;
        }
        
        const menuPanel = document.getElementById('menuPanel');
        if (menuPanel.classList.contains('hidden')) {
            // showPanel will handle the dirty check for settings
            showPanel('menuPanel');
        }
    }
});

