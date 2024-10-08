<script lang="ts">
	import { onMount } from 'svelte';
	import QrScanner from 'qr-scanner';
	import type { ClassroomWithData, Student } from '$lib/types/classroom';
	import type { User } from '$lib/types/user';

	export let classroom: ClassroomWithData;
	export let currentUser: User;

	let studentId: string = '';
	let neuronsToSend: number = 0;
	let qrScanner: QrScanner | null = null;
	let showScanner = false;
	let cameraError = '';
	let videoSource: HTMLVideoElement;
	let loading = false;

	const denominations = [1, 2, 5, 10, 20];

	onMount(() => {
		if (!videoSource) {
			console.error('Video element not found on mount');
		}
		return () => {
			stopScanner();
		};
	});

	async function toggleScanner() {
		if (showScanner) {
			stopScanner();
		} else {
			await startScanner();
		}
	}

	async function startScanner() {
		try {
			loading = true;
			cameraError = '';

			if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
				throw new Error('getUserMedia is not supported in this browser');
			}

			if (!videoSource) {
				throw new Error('Video element not found');
			}

			let stream;
			try {
				stream = await navigator.mediaDevices.getUserMedia({
					video: { facingMode: 'environment' }
				});
			} catch (err) {
				console.warn('Failed to get environment-facing camera:', err);
				stream = await navigator.mediaDevices.getUserMedia({
					video: true
				});
			}

			videoSource.srcObject = stream;
			await new Promise((resolve) => {
				videoSource.onloadedmetadata = () => {
					resolve(null);
				};
			});
			await videoSource.play();

			await new Promise((resolve) => setTimeout(resolve, 1000)); // 1 second delay

			qrScanner = new QrScanner(
				videoSource,
				(result) => {
					studentId = result.data;
					stopScanner();
				},
				{ highlightScanRegion: true, highlightCodeOutline: true }
			);

			await qrScanner.start();
			showScanner = true;
		} catch (error: any) {
			console.error('Detailed error starting scanner:', error);
			if (error instanceof DOMException && error.name === 'NotAllowedError') {
				cameraError = 'Camera access was denied. Please grant permission and try again.';
			} else if (error instanceof DOMException && error.name === 'NotFoundError') {
				cameraError = 'No camera found on this device.';
			} else {
				cameraError = `Failed to access camera: ${error.message}. Please ensure you have given permission and are using a supported browser.`;
			}
		} finally {
			loading = false;
		}
	}

	function stopScanner() {
		if (qrScanner) {
			qrScanner.stop();
			qrScanner.destroy();
			qrScanner = null;
		}
		if (videoSource && videoSource.srcObject) {
			const tracks = (videoSource.srcObject as MediaStream).getTracks();
			tracks.forEach((track) => track.stop());
		}
		showScanner = false;
	}

	function addNeurons(amount: number) {
		neuronsToSend += amount;
	}

	function resetNeurons() {
		neuronsToSend = 0;
	}

	async function sendNeurons() {
		if (!studentId || neuronsToSend <= 0) {
			alert('Please enter a valid student ID and neuron amount');
			return;
		}

		console.log(`Sending ${neuronsToSend} neurons to student ${studentId}`);

		const studentIndex = classroom.students.findIndex((s) => s.user.id.toString() === studentId);
		if (studentIndex !== -1) {
			classroom.students[studentIndex].neurons += neuronsToSend;
			classroom.availableNeurons -= neuronsToSend;
		}

		studentId = '';
		neuronsToSend = 0;
	}
</script>

<div class="min-h-screen bg-gradient-to-br from-blue-400 to-purple-500 px-4 py-12 sm:px-6 lg:px-8">
	<div class="mx-auto max-w-4xl">
		<div class="overflow-hidden rounded-2xl bg-white shadow-2xl">
			<div class="bg-gradient-to-r from-blue-600 to-purple-600 px-6 py-8 sm:px-8">
				<h1 class="text-3xl font-extrabold text-white">{classroom.name}</h1>
				<p class="mt-2 text-xl font-semibold text-blue-100">Teacher View</p>
				<div
					class="mt-4 inline-flex items-center rounded-full bg-blue-500 bg-opacity-20 px-4 py-2 text-sm font-medium text-white"
				>
					<svg class="mr-2 h-5 w-5" fill="currentColor" viewBox="0 0 20 20">
						<path
							d="M13 6a3 3 0 11-6 0 3 3 0 016 0zM18 8a2 2 0 11-4 0 2 2 0 014 0zM14 15a4 4 0 00-8 0v3h8v-3zM6 8a2 2 0 11-4 0 2 2 0 014 0zM16 18v-3a5.972 5.972 0 00-.75-2.906A3.005 3.005 0 0119 15v3h-3zM4.75 12.094A5.973 5.973 0 004 15v3H1v-3a3 3 0 013.75-2.906z"
						/>
					</svg>
					Available Neurons: {classroom.availableNeurons}
				</div>
			</div>
			<div class="px-6 py-8 sm:px-8">
				<div class="mb-6">
					<label for="student-id" class="block text-lg font-medium text-gray-700">Student ID</label>
					<div class="mt-2 flex items-center space-x-4">
						<div class="relative flex-grow">
							<input
								type="text"
								id="student-id"
								bind:value={studentId}
								class="block w-full rounded-lg border-gray-300 pr-10 text-lg focus:border-purple-500 focus:ring-purple-500"
								placeholder="Enter student ID"
							/>
							{#if studentId}
								<button
									on:click={() => (studentId = '')}
									class="absolute inset-y-0 right-0 flex items-center pr-3 text-gray-400 hover:text-gray-500"
								>
									<svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											stroke-width="2"
											d="M6 18L18 6M6 6l12 12"
										/>
									</svg>
								</button>
							{/if}
						</div>
						<button
							on:click={toggleScanner}
							class="inline-flex items-center justify-center rounded-lg bg-purple-600 px-4 py-2 text-base font-medium text-white shadow-sm transition-all duration-300 ease-in-out hover:bg-purple-700 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:ring-offset-2 {showScanner
								? 'bg-red-600 hover:bg-red-700'
								: ''}"
						>
							<svg class="mr-2 h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M12 4v1m6 11h2m-6 0h-2v4m0-11v-4m6 6v4m-6-4h6m-6 4h6m6-4h2m-2 4h2"
								/>
							</svg>
							{showScanner ? 'Stop Scan' : 'Scan QR Code'}
						</button>
					</div>
					{#if cameraError}
						<p class="mt-2 text-sm text-red-600">{cameraError}</p>
					{/if}
					{#if loading}
						<p class="mt-2 text-sm text-blue-600">Initializing camera...</p>
					{/if}
				</div>
				<div
					class="mb-6 overflow-hidden rounded-lg bg-gray-100 p-4"
					style="display: {showScanner ? 'block' : 'none'}"
				>
					<video bind:this={videoSource} width="100%" height="300" class="rounded-lg"></video>
				</div>
				<div class="mb-6">
					<label class="block text-lg font-medium text-gray-700">Neurons to Send</label>
					<div class="mt-2 flex items-center space-x-2">
						<span class="text-3xl font-bold text-purple-600">{neuronsToSend}</span>
						<button
							on:click={resetNeurons}
							class="ml-2 inline-flex items-center rounded-full border border-transparent bg-purple-100 p-2 text-purple-600 hover:bg-purple-200 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:ring-offset-2"
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
								class="inline-flex items-center rounded-full border border-transparent bg-purple-600 px-6 py-3 text-lg font-medium text-white shadow-sm hover:bg-purple-700 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:ring-offset-2"
							>
								+{denom}
							</button>
						{/each}
					</div>
				</div>
				<button
					on:click={sendNeurons}
					class="inline-flex w-full items-center justify-center rounded-md border border-transparent bg-gradient-to-r from-blue-600 to-purple-600 px-6 py-4 text-xl font-medium text-white shadow-sm hover:from-blue-700 hover:to-purple-700 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:ring-offset-2"
				>
					<svg class="-ml-1 mr-3 h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M13 10V3L4 14h7v7l9-11h-7z"
						/>
					</svg>
					Send Neurons
				</button>
			</div>
		</div>
	</div>
</div>
