// Calculate total services on page load
document.addEventListener('DOMContentLoaded', function() {
    let totalServices = 0;
    document.querySelectorAll('#tsl-tbody tr').forEach(row => {
        const services = parseInt(row.cells[4].textContent) || 0;
        totalServices += services;
    });
    document.getElementById('total-services').textContent = totalServices.toLocaleString();

    // Populate type filter
    const types = new Set();
    document.querySelectorAll('#tsl-tbody tr').forEach(row => {
        const type = row.getAttribute('data-type');
        if (type) types.add(type);
    });
    const filterSelect = document.getElementById('filter-type');
    Array.from(types).sort().forEach(type => {
        const option = document.createElement('option');
        option.value = type;
        option.textContent = type;
        filterSelect.appendChild(option);
    });
});

// Search functionality
document.getElementById('search').addEventListener('input', filterTable);
document.getElementById('filter-type').addEventListener('change', filterTable);

function filterTable() {
    const searchTerm = document.getElementById('search').value.toLowerCase();
    const filterType = document.getElementById('filter-type').value;
    const rows = document.querySelectorAll('#tsl-tbody tr');
    let visibleCount = 0;

    rows.forEach(row => {
        const text = row.textContent.toLowerCase();
        const type = row.getAttribute('data-type');
        const matchesSearch = text.includes(searchTerm);
        const matchesType = !filterType || type === filterType;

        if (matchesSearch && matchesType) {
            row.style.display = '';
            visibleCount++;
        } else {
            row.style.display = 'none';
        }
    });

    document.getElementById('no-results').style.display = visibleCount === 0 ? 'block' : 'none';
}

// Table sorting
let sortColumn = -1;
let sortAscending = true;

function sortTable(columnIndex) {
    const table = document.getElementById('tsl-table');
    const tbody = document.getElementById('tsl-tbody');
    const rows = Array.from(tbody.querySelectorAll('tr'));

    // Update sort direction
    if (sortColumn === columnIndex) {
        sortAscending = !sortAscending;
    } else {
        sortAscending = true;
        sortColumn = columnIndex;
    }

    // Remove sort classes from all headers
    table.querySelectorAll('th').forEach(th => {
        th.classList.remove('sort-asc', 'sort-desc');
    });

    // Add sort class to current header
    const header = table.querySelectorAll('th')[columnIndex];
    header.classList.add(sortAscending ? 'sort-asc' : 'sort-desc');

    // Sort rows
    rows.sort((a, b) => {
        let aValue = a.cells[columnIndex].textContent.trim();
        let bValue = b.cells[columnIndex].textContent.trim();

        // Extract numeric values from badges
        if (columnIndex === 0) {
            aValue = a.getAttribute('data-territory') || aValue;
            bValue = b.getAttribute('data-territory') || bValue;
        }

        // Try to parse as numbers
        const aNum = parseFloat(aValue.replace(/[^0-9.-]/g, ''));
        const bNum = parseFloat(bValue.replace(/[^0-9.-]/g, ''));

        if (!isNaN(aNum) && !isNaN(bNum)) {
            return sortAscending ? aNum - bNum : bNum - aNum;
        }

        // String comparison
        return sortAscending ? 
            aValue.localeCompare(bValue) : 
            bValue.localeCompare(aValue);
    });

    // Reorder rows in DOM
    rows.forEach(row => tbody.appendChild(row));
}

// Theme toggle
function toggleTheme() {
    const html = document.documentElement;
    const currentTheme = html.getAttribute('data-theme');
    const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
    html.setAttribute('data-theme', newTheme);
    localStorage.setItem('theme', newTheme);
}

// Load saved theme
document.addEventListener('DOMContentLoaded', function() {
    const savedTheme = localStorage.getItem('theme') || 
        (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
    document.documentElement.setAttribute('data-theme', savedTheme);
});
