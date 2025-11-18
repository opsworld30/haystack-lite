let allFiles = [];
let selectedFiles = new Set();
let currentPage = 1;
let pageSize = 20;
let totalPages = 1;
let totalFiles = 0;
let searchKeyword = '';

document.addEventListener('DOMContentLoaded', () => {
    loadFiles();
    setupUpload();
    setupSearch();
});

function setupUpload() {
    const uploadBox = document.getElementById('uploadBox');
    const fileInput = document.getElementById('fileInput');

    uploadBox.addEventListener('dragover', (e) => {
        e.preventDefault();
        uploadBox.classList.add('drag-over');
    });

    uploadBox.addEventListener('dragleave', () => {
        uploadBox.classList.remove('drag-over');
    });

    uploadBox.addEventListener('drop', (e) => {
        e.preventDefault();
        uploadBox.classList.remove('drag-over');
        const files = e.dataTransfer.files;
        uploadFiles(files);
    });

    fileInput.addEventListener('change', (e) => {
        uploadFiles(e.target.files);
    });
}

function setupSearch() {
    const searchInput = document.getElementById('searchInput');
    searchInput.addEventListener('input', (e) => {
        searchKeyword = e.target.value.toLowerCase();
        currentPage = 1;
        loadFiles();
    });
}

async function uploadFiles(files) {
    if (files.length === 0) return;

    const progressDiv = document.getElementById('uploadProgress');
    const progressFill = document.getElementById('progressFill');
    const progressText = document.getElementById('progressText');
    
    progressDiv.style.display = 'block';
    
    let completed = 0;
    const total = files.length;

    for (const file of files) {
        const formData = new FormData();
        formData.append('file', file);

        try {
            const response = await fetch('/file', {
                method: 'POST',
                body: formData
            });

            if (!response.ok) {
                throw new Error(`上传失败: ${file.name}`);
            }

            completed++;
            const percent = (completed / total * 100).toFixed(0);
            progressFill.style.width = percent + '%';
            progressText.textContent = `上传中... ${completed}/${total} (${percent}%)`;

        } catch (error) {
            console.error('上传错误:', error);
            alert(`上传失败: ${file.name}`);
        }
    }

    progressText.textContent = `上传完成! ${completed}/${total}`;
    setTimeout(() => {
        progressDiv.style.display = 'none';
        progressFill.style.width = '0%';
        document.getElementById('fileInput').value = '';
        loadFiles();
    }, 1500);
}

async function loadFiles() {
    try {
        let url = `/files?page=${currentPage}&page_size=${pageSize}`;
        const response = await fetch(url);
        const data = await response.json();
        
        allFiles = data.files || [];
        totalFiles = data.total || 0;
        totalPages = data.total_page || 1;
        currentPage = data.page || 1;
        
        if (searchKeyword) {
            allFiles = allFiles.filter(f => 
                f.filename.toLowerCase().includes(searchKeyword)
            );
        }
        
        renderFiles(allFiles);
        updatePagination();
        updateStats();
    } catch (error) {
        console.error('加载文件列表失败:', error);
        document.getElementById('fileTableBody').innerHTML = 
            '<tr><td colspan="7" class="loading">加载失败</td></tr>';
    }
}

function renderFiles(files) {
    const tbody = document.getElementById('fileTableBody');
    
    if (files.length === 0) {
        tbody.innerHTML = '<tr><td colspan="7" class="loading">暂无文件</td></tr>';
        return;
    }

    tbody.innerHTML = files.map(file => `
        <tr>
            <td><input type="checkbox" class="file-checkbox" value="${file.id}" onchange="updateSelection()"></td>
            <td>${file.id}</td>
            <td class="file-name">${escapeHtml(file.filename)}</td>
            <td><span class="file-type">${getFileTypeLabel(file.mime_type)}</span></td>
            <td>${formatSize(file.size)}</td>
            <td>${formatTime(file.created_at)}</td>
            <td class="actions">
                <button class="btn btn-primary btn-sm" onclick="previewFile(${file.id})">预览</button>
                <button class="btn btn-secondary btn-sm" onclick="downloadFile(${file.id}, '${escapeHtml(file.filename)}')">下载</button>
                <button class="btn btn-danger btn-sm" onclick="deleteFile(${file.id})">删除</button>
            </td>
        </tr>
    `).join('');
}

function updateStats() {
    const totalSize = allFiles.reduce((sum, f) => sum + f.size, 0);
    
    document.getElementById('totalFiles').textContent = `文件总数: ${totalFiles}`;
    document.getElementById('totalSize').textContent = `当前页大小: ${formatSize(totalSize)}`;
}

function updatePagination() {
    const pagination = document.getElementById('pagination');
    const pageInfo = document.getElementById('pageInfo');
    const prevBtn = document.getElementById('prevPage');
    const nextBtn = document.getElementById('nextPage');
    
    if (totalPages > 1) {
        pagination.style.display = 'flex';
        pageInfo.textContent = `第 ${currentPage} 页 / 共 ${totalPages} 页`;
        prevBtn.disabled = currentPage <= 1;
        nextBtn.disabled = currentPage >= totalPages;
    } else {
        pagination.style.display = 'none';
    }
}

function changePage(delta) {
    const newPage = currentPage + delta;
    if (newPage >= 1 && newPage <= totalPages) {
        currentPage = newPage;
        loadFiles();
    }
}

function updateSelection() {
    selectedFiles.clear();
    document.querySelectorAll('.file-checkbox:checked').forEach(cb => {
        selectedFiles.add(parseInt(cb.value));
    });
    document.getElementById('deleteBtn').disabled = selectedFiles.size === 0;
}

function toggleSelectAll() {
    const selectAll = document.getElementById('selectAll');
    document.querySelectorAll('.file-checkbox').forEach(cb => {
        cb.checked = selectAll.checked;
    });
    updateSelection();
}

async function previewFile(id) {
    const modal = document.getElementById('previewModal');
    const content = document.getElementById('previewContent');
    
    const file = allFiles.find(f => f.id === id);
    if (!file) return;

    const mimeType = file.mime_type;
    const previewUrl = `/file/${id}/preview`;

    if (mimeType.startsWith('image/')) {
        content.innerHTML = `<img src="${previewUrl}" alt="${escapeHtml(file.filename)}">`;
    } else if (mimeType.startsWith('video/')) {
        content.innerHTML = `<video controls src="${previewUrl}"></video>`;
    } else if (mimeType.startsWith('audio/')) {
        content.innerHTML = `<audio controls src="${previewUrl}"></audio>`;
    } else if (mimeType === 'application/pdf') {
        content.innerHTML = `<iframe src="${previewUrl}"></iframe>`;
    } else if (mimeType.startsWith('text/')) {
        try {
            const response = await fetch(previewUrl);
            const text = await response.text();
            content.innerHTML = `<pre>${escapeHtml(text)}</pre>`;
        } catch (error) {
            content.innerHTML = '<p style="text-align: center; color: #9ca3af; padding: 40px;">预览失败</p>';
        }
    } else {
        content.innerHTML = '<p style="text-align: center; color: #9ca3af; padding: 40px;">该文件类型不支持预览</p>';
    }

    modal.classList.add('show');
}

function closePreview() {
    document.getElementById('previewModal').classList.remove('show');
}

function downloadFile(id, filename) {
    const a = document.createElement('a');
    a.href = `/file/${id}`;
    a.download = filename;
    a.click();
}

async function deleteFile(id) {
    if (!confirm('确定要删除这个文件吗？')) return;

    try {
        const response = await fetch(`/file/${id}`, {
            method: 'DELETE'
        });

        if (response.ok) {
            loadFiles();
        } else {
            alert('删除失败');
        }
    } catch (error) {
        console.error('删除错误:', error);
        alert('删除失败');
    }
}

async function deleteSelected() {
    if (selectedFiles.size === 0) return;
    if (!confirm(`确定要删除选中的 ${selectedFiles.size} 个文件吗？`)) return;

    try {
        const response = await fetch('/files/batch/delete', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                ids: Array.from(selectedFiles)
            })
        });

        if (response.ok) {
            selectedFiles.clear();
            document.getElementById('selectAll').checked = false;
            loadFiles();
        } else {
            alert('批量删除失败');
        }
    } catch (error) {
        console.error('批量删除错误:', error);
        alert('批量删除失败');
    }
}

function refreshFiles() {
    loadFiles();
}

function formatSize(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return (bytes / Math.pow(k, i)).toFixed(2) + ' ' + sizes[i];
}

function formatTime(timestamp) {
    const date = new Date(timestamp * 1000);
    return date.toLocaleString('zh-CN');
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function getFileTypeLabel(mimeType) {
    if (mimeType.startsWith('image/')) return 'image';
    if (mimeType.startsWith('video/')) return 'video';
    if (mimeType.startsWith('audio/')) return 'audio';
    if (mimeType === 'application/pdf') return 'pdf';
    if (mimeType.startsWith('text/')) return 'text';
    if (mimeType.includes('zip') || mimeType.includes('tar') || mimeType.includes('gzip')) return 'archive';
    if (mimeType.includes('json') || mimeType.includes('xml')) return 'data';
    return 'file';
}
