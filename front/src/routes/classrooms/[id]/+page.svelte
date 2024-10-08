<!-- src/routes/classroom/[id].svelte -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { currentUser } from '$lib/stores/userStore';
	import type { ClassroomWithData } from '$lib/types/classroom';
	import type { User } from '$lib/types/user';
	import TeacherClassroom from '$lib/components/TeacherClassroom.svelte';
	import StudentClassroom from '$lib/components/StudentClassroom.svelte';

	let classroom: ClassroomWithData | null = null;
	let user: User | null = null;

	currentUser.subscribe((value) => {
		user = value;
	});

	onMount(async () => {
		const classroomId = $page.params.id;
		// Fetch classroom data based on the ID
		// This is a placeholder, replace with your actual API call
		classroom = await fetchClassroomData(classroomId);
	});

	async function fetchClassroomData(id: string): Promise<ClassroomWithData> {
		// In a real application, you would fetch this data from an API
		// For now, we're returning mock data
		return {
			id: parseInt(id),
			name: 'Advanced Mathematics',
			teacherID: 101,
			availableNeurons: 1000,
			createdAt: new Date('2023-01-01T00:00:00Z'),
			teacher: {
				id: 101,
				name: 'Dr. Jane Smith',
				email: 'jane.smith@example.com',
				role: 'teacher',
				createdAt: new Date('2022-01-01T00:00:00Z')
			},
			students: [
				{
					user: {
						id: 201,
						name: 'John Doe',
						email: 'john.doe@example.com',
						role: 'student',
						createdAt: new Date('2023-01-15T00:00:00Z')
					},
					neurons: 150
				},
				{
					user: {
						id: 202,
						name: 'Bob Williams',
						email: 'bob.williams@example.com',
						role: 'student',
						createdAt: new Date('2023-01-11T00:00:00Z')
					},
					neurons: 200
				},
				{
					user: {
						id: 203,
						name: 'Alice Johnson',
						email: 'alice.johnson@example.com',
						role: 'student',
						createdAt: new Date('2023-01-20T00:00:00Z')
					},
					neurons: 175
				},
				{
					user: {
						id: 204,
						name: 'Emma Brown',
						email: 'emma.brown@example.com',
						role: 'student',
						createdAt: new Date('2023-01-18T00:00:00Z')
					},
					neurons: 225
				},
				{
					user: {
						id: 205,
						name: 'Michael Lee',
						email: 'michael.lee@example.com',
						role: 'student',
						createdAt: new Date('2023-01-22T00:00:00Z')
					},
					neurons: 100
				},
				{
					user: {
						id: 206,
						name: 'Sarah Davis',
						email: 'sarah.davis@example.com',
						role: 'student',
						createdAt: new Date('2023-01-25T00:00:00Z')
					},
					neurons: 180
				},
				{
					user: {
						id: 207,
						name: 'David Wilson',
						email: 'david.wilson@example.com',
						role: 'student',
						createdAt: new Date('2023-01-28T00:00:00Z')
					},
					neurons: 190
				}
			]
		};
	}
</script>

{#if classroom && user}
	{#if user.role === 'teacher'}
		<TeacherClassroom {classroom} currentUser={user} />
	{:else if user.role === 'student'}
		<StudentClassroom {classroom} />
	{:else}
		<p>Unauthorized access.</p>
	{/if}
{:else}
	<p>Loading classroom data...</p>
{/if}
