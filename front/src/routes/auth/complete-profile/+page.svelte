<script lang="ts">
	import { enhance } from '$app/forms';
	import { page } from '$app/stores';

	let name = '';
	let role = 'student'; // Default role

	$: userId = $page.url.searchParams.get('userId') || '';
	$: email = $page.url.searchParams.get('email') || '';
	$: googleId = $page.url.searchParams.get('googleId') || '';

	$: ({ message } = $page.form || {});
</script>

<div class="mx-auto mt-8 max-w-md rounded-lg bg-white p-6 shadow-md">
	<h1 class="mb-2 text-2xl font-bold text-gray-800">Complete Your Profile</h1>
	<p class="mb-6 text-gray-600">Please enter your details to complete your registration.</p>

	<form method="POST" use:enhance class="space-y-4">
		<input type="hidden" name="userId" value={userId} />
		<input type="hidden" name="email" value={email} />
		<input type="hidden" name="googleId" value={googleId} />

		<div>
			<label for="name" class="mb-1 block text-sm font-medium text-gray-700">Full Name</label>
			<input
				type="text"
				id="name"
				name="name"
				bind:value={name}
				required
				class="w-full rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
			/>
		</div>

		<div>
			<label for="role" class="mb-1 block text-sm font-medium text-gray-700">Role</label>
			<select
				id="role"
				name="role"
				bind:value={role}
				required
				class="w-full rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
			>
				<option value="student">Student</option>
				<option value="teacher">Teacher</option>
			</select>
		</div>

		{#if message}
			<p class="text-sm text-red-600">{message}</p>
		{/if}

		<button
			type="submit"
			class="w-full rounded-md bg-blue-600 px-4 py-2 text-white transition-colors hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
		>
			Complete Registration
		</button>
	</form>
</div>
