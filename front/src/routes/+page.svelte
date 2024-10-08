<script lang="ts">
	import { currentUser } from '$lib/stores/userStore';
</script>

<div>
	{#if $currentUser}
		<h1>Welcome, {$currentUser.name}!</h1>
		<p>Email: {$currentUser.email}</p>
		<p>Role: {$currentUser.role}</p>
		<!-- Add any other user properties you want to display -->
	{:else}
		<p>Loading user information...</p>
	{/if}
</div>

<div>hola</div>
