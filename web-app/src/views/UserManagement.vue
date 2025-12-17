<template>
  <div class="min-h-screen bg-gray-50">
    <!-- Navigation -->
    <NavBar />

    <!-- Main Content -->
    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <!-- Page Header -->
      <div class="mb-8">
        <h2 class="text-2xl font-bold text-gray-900">User Management</h2>
        <p class="mt-1 text-sm text-gray-600">Manage users and assign server permissions</p>
      </div>

      <!-- Placeholder Content -->
      <div class="bg-white rounded-lg shadow p-12 text-center">
        <svg class="mx-auto h-16 w-16 text-gray-400 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
        </svg>
        <h3 class="text-lg font-medium text-gray-900 mb-2">User Management Coming Soon</h3>
        <p class="text-gray-600 max-w-md mx-auto">
          This feature will allow you to manage users, assign roles, and configure server permissions.
          Implementation is planned for a future release.
        </p>
      </div>
    </main>
  </div>
</template>

<script setup>
import NavBar from '@/components/layout/NavBar.vue'
</script>
