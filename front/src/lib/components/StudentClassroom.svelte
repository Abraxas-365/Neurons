<script lang="ts">
	import { onMount } from 'svelte';
	import QRCode from 'qrcode';
	import { currentUser } from '$lib/stores/userStore';
	import type { ClassroomWithData, Student } from '$lib/types/classroom';
	import type { User } from '$lib/types/user';

	export let classroom: ClassroomWithData;

	let user: User | null;
	currentUser.subscribe((value) => {
		user = value;
	});

	let qrCodeDataURL: string = '';
	let studentNeurons: number = 0;
	let neuronsToReturn: number = 0;

	const denominations = [1, 2, 5, 10, 20];

	$: if (user && classroom) {
		const currentStudent = classroom.students.find((student) => student.user.id === user!.id);
		if (currentStudent) {
			studentNeurons = currentStudent.neurons;
		}
	}

	onMount(async () => {
		if (user) {
			try {
				qrCodeDataURL = await QRCode.toDataURL(user.id.toString(), {
					width: 200,
					margin: 2
				});
			} catch (err) {
				console.error('Error generating QR code:', err);
			}
		}
	});

	function addNeurons(amount: number) {
		neuronsToReturn += amount;
	}

	function resetNeurons() {
		neuronsToReturn = 0;
	}

	async function returnNeurons() {
		if (neuronsToReturn <= 0 || neuronsToReturn > studentNeurons) {
			alert('Please enter a valid number of neurons to return');
			return;
		}

		try {
			const response = await fetch(`/api/classrooms/${classroom.id}/return-neurons`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({ amount: neuronsToReturn })
			});

			if (!response.ok) {
				throw new Error('Failed to return neurons');
			}

			studentNeurons -= neuronsToReturn;
			neuronsToReturn = 0;
			alert('Neurons returned successfully!');
		} catch (error) {
			console.error('Error returning neurons:', error);
			alert('Failed to return neurons. Please try again.');
		}
	}
</script>

{#if user}
	<div
		class="min-h-screen bg-gradient-to-br from-blue-100 to-indigo-100 px-4 py-12 sm:px-6 lg:px-8"
	>
		<div class="mx-auto max-w-3xl">
			<div class="overflow-hidden rounded-lg bg-white shadow-xl">
				<div class="bg-indigo-600 px-4 py-5 sm:px-6">
					<h1 class="text-2xl font-bold text-white">{classroom.name}</h1>
					<p class="mt-1 max-w-2xl text-sm text-indigo-100">Teacher: {classroom.teacher.name}</p>
				</div>
				<div class="px-4 py-5 sm:p-6">
					<div class="text-center">
						<h2 class="text-3xl font-extrabold text-gray-900">Your Neurons</h2>
						<p class="mt-4 text-5xl font-bold text-indigo-600">{studentNeurons}</p>
					</div>
					<div class="mt-10 flex flex-col items-center justify-center">
						{#if qrCodeDataURL}
							<div class="rounded-lg bg-white p-4 shadow-md">
								<img src={qrCodeDataURL} alt="Student ID QR Code" class="mx-auto" />
								<p class="mt-2 text-center text-sm text-gray-500">Your Student ID QR Code</p>
							</div>
							<p class="mt-4 text-lg font-semibold text-gray-700">User ID: {user.id}</p>
						{:else}
							<p class="text-gray-500">Loading QR code...</p>
						{/if}
					</div>
					<div class="mt-10">
						<h3 class="text-lg font-medium text-gray-900">Return Neurons to Classroom</h3>
						<div class="mt-4 flex items-center space-x-2">
							<span class="text-2xl font-bold text-indigo-600">{neuronsToReturn}</span>
							<button
								on:click={resetNeurons}
								class="ml-2 inline-flex items-center rounded-full border border-transparent bg-indigo-100 p-2 text-indigo-600 hover:bg-indigo-200 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2"
							>
								<svg class="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
									/>
								</svg>
							</button>
						</div>
						<div class="mt-4 flex flex-wrap gap-2">
							{#each denominations as denom}
								<button
									on:click={() => addNeurons(denom)}
									class="inline-flex items-center rounded-full border border-transparent bg-indigo-600 px-6 py-3 text-lg font-medium text-white shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2"
									disabled={neuronsToReturn + denom > studentNeurons}
								>
									+{denom}
								</button>
							{/each}
						</div>
						<button
							on:click={returnNeurons}
							class="mt-6 inline-flex w-full items-center justify-center rounded-md border border-transparent bg-indigo-600 px-6 py-3 text-base font-medium text-white shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2"
							disabled={neuronsToReturn <= 0 || neuronsToReturn > studentNeurons}
						>
							Return Neurons
						</button>
					</div>
				</div>
			</div>
		</div>
	</div>
{:else}
	<p>Please log in to view your classroom.</p>
{/if}
