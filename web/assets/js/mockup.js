// Modal Management Functions

// Add Server Modal
function openAddModal() {
    document.getElementById('addModal').classList.remove('hidden');
}

function closeAddModal() {
    document.getElementById('addModal').classList.add('hidden');
}

// Edit Server Modal
function openEditModal(serverId) {
    // In a real app, this would fetch server data and populate the form
    console.log('Editing server:', serverId);
    document.getElementById('editModal').classList.remove('hidden');
}

function closeEditModal() {
    document.getElementById('editModal').classList.add('hidden');
}

// Delete Confirmation Modal
function openDeleteModal(serverName) {
    document.getElementById('deleteServerName').textContent = serverName;
    document.getElementById('deleteModal').classList.remove('hidden');
}

function closeDeleteModal() {
    document.getElementById('deleteModal').classList.add('hidden');
}

// Close modals when clicking outside
window.addEventListener('click', function(event) {
    const addModal = document.getElementById('addModal');
    const editModal = document.getElementById('editModal');
    const deleteModal = document.getElementById('deleteModal');

    if (event.target === addModal) {
        closeAddModal();
    }
    if (event.target === editModal) {
        closeEditModal();
    }
    if (event.target === deleteModal) {
        closeDeleteModal();
    }
});

// Close modals with Escape key
document.addEventListener('keydown', function(event) {
    if (event.key === 'Escape') {
        closeAddModal();
        closeEditModal();
        closeDeleteModal();
    }
});

// Auth Type Toggle for Add Modal
document.addEventListener('DOMContentLoaded', function() {
    // Get radio buttons for Add modal
    const authRadios = document.querySelectorAll('input[name="auth_type"]');
    const basicAuthDiv = document.getElementById('basicAuth');
    const bearerAuthDiv = document.getElementById('bearerAuth');
    const oauthAuthDiv = document.getElementById('oauthAuth');

    authRadios.forEach(radio => {
        radio.addEventListener('change', function() {
            // Hide all auth config sections
            basicAuthDiv.classList.add('hidden');
            bearerAuthDiv.classList.add('hidden');
            oauthAuthDiv.classList.add('hidden');

            // Show the selected auth config section
            if (this.value === 'basic') {
                basicAuthDiv.classList.remove('hidden');
            } else if (this.value === 'bearer') {
                bearerAuthDiv.classList.remove('hidden');
            } else if (this.value === 'oauth') {
                oauthAuthDiv.classList.remove('hidden');
            }
        });
    });
});

// Toggle Switch Animation (optional enhancement)
function toggleServer(serverId) {
    console.log('Toggling server:', serverId);
    // In a real app, this would make an API call to toggle the server status
    alert('This is a mockup - no actual toggle performed');
}

// Form Submission Handlers (mockup alerts)
document.addEventListener('DOMContentLoaded', function() {
    // Prevent form submissions and show mockup alerts
    const forms = document.querySelectorAll('form');
    forms.forEach(form => {
        form.addEventListener('submit', function(event) {
            event.preventDefault();
            alert('This is a static mockup. In a real application, this would submit data to the API.');
        });
    });
});

// Logout function
function logout() {
    if (confirm('Are you sure you want to logout?')) {
        window.location.href = 'login.html';
    }
}
