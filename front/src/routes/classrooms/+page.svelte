<script lang="ts">
	import { onMount } from 'svelte';
	import ClassroomCard from '$lib/components/ClassroomCard.svelte';
	import type { ClassroomWithData } from '$lib/types/classroom';
	import type { User } from '$lib/types/user';
	import { currentUser } from '$lib/stores/userStore';
	import { api } from '$lib/api';

	let user: User;
	let isTeacher: boolean = false;
	let classrooms: ClassroomWithData[] = [];
	let loading: boolean = true;
	let error: string | null = null;

	currentUser.subscribe((value) => {
		user = value;
		isTeacher = user?.role === 'teacher';
	});

	onMount(async () => {
		try {
			const response = await api.get('/classrooms/user');
			classrooms = response.data;
			loading = false;
		} catch (err) {
			console.error('Error fetching classrooms:', err);
			error = 'Failed to load classrooms. Please try again later.';
			loading = false;
		}
	});

	// Function to handle creating a new classroom
	async function createNewClassroom() {
		// Implement the logic to create a new classroom
		console.log('Create new classroom');
		// This could open a modal or navigate to a new page for classroom creation
	}
</script>

<div
	class="min-h-screen bg-gradient-to-br from-indigo-100 to-purple-100 px-4 py-12 sm:px-6 lg:px-8"
>
	<div class="mx-auto max-w-7xl">
		<div class="text-center">
			<h1 class="text-4xl font-extrabold text-gray-900 sm:text-5xl md:text-6xl">
				<span class="block">Your Classrooms</span>
				<span class="mt-2 block text-indigo-600">Explore and Learn</span>
			</h1>
			<p
				class="mx-auto mt-3 max-w-md text-base text-gray-500 sm:text-lg md:mt-5 md:max-w-3xl md:text-xl"
			>
				Discover your learning spaces, connect with teachers, and track your progress in one place.
			</p>
		</div>

		{#if isTeacher}
			<div class="mt-8 flex justify-center">
				<button
					on:click={createNewClassroom}
					class="inline-flex items-center rounded-md border border-transparent bg-indigo-600 px-6 py-3 text-base font-medium text-white shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2"
				>
					<svg
						class="-ml-1 mr-2 h-5 w-5"
						xmlns="http://www.w3.org/2000/svg"
						viewBox="0 0 20 20"
						fill="currentColor"
						aria-hidden="true"
					>
						<path
							fill-rule="evenodd"
							d="M10 5a1 1 0 011 1v3h3a1 1 0 110 2h-3v3a1 1 0 11-2 0v-3H6a1 1 0 110-2h3V6a1 1 0 011-1z"
							clip-rule="evenodd"
						/>
					</svg>
					Create New Classroom
				</button>
			</div>
		{/if}

		<div class="mt-12">
			{#if loading}
				<p class="text-center">Loading classrooms...</p>
			{:else if error}
				<p class="text-center text-red-500">{error}</p>
			{:else if classrooms.length === 0}
				<p class="text-center">No classrooms found.</p>
			{:else}
				<div class="grid grid-cols-1 gap-8 sm:grid-cols-2 lg:grid-cols-3">
					{#each classrooms as classroom (classroom.id)}
						<ClassroomCard {classroom} />
					{/each}
				</div>
			{/if}
		</div>
	</div>
</div>
