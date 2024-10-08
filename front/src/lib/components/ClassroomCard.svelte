<script lang="ts">
	import type { ClassroomWithData } from '$lib/types/classroom';

	export let classroom: ClassroomWithData;
</script>

<div
	class="flex flex-col justify-between overflow-hidden rounded-lg bg-white shadow-lg transition-all duration-300 hover:scale-105 hover:shadow-xl"
>
	<div class="p-6">
		<h2 class="mb-2 text-2xl font-bold text-gray-900">{classroom.name}</h2>
		<p class="mb-4 text-sm text-gray-600">Teacher: {classroom.teacher.name}</p>
		<div class="mb-4 flex items-center justify-between">
			<div class="flex items-center">
				<svg
					xmlns="http://www.w3.org/2000/svg"
					class="mr-2 h-5 w-5 text-yellow-500"
					viewBox="0 0 20 20"
					fill="currentColor"
				>
					<path
						d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"
					/>
				</svg>
				<span class="font-semibold text-gray-700">{classroom.availableNeurons} Neurons</span>
			</div>
			<div class="flex items-center">
				<svg
					xmlns="http://www.w3.org/2000/svg"
					class="mr-2 h-5 w-5 text-blue-500"
					viewBox="0 0 20 20"
					fill="currentColor"
				>
					<path
						d="M13 6a3 3 0 11-6 0 3 3 0 016 0zM18 8a2 2 0 11-4 0 2 2 0 014 0zM14 15a4 4 0 00-8 0v3h8v-3zM6 8a2 2 0 11-4 0 2 2 0 014 0zM16 18v-3a5.972 5.972 0 00-.75-2.906A3.005 3.005 0 0119 15v3h-3zM4.75 12.094A5.973 5.973 0 004 15v3H1v-3a3 3 0 013.75-2.906z"
					/>
				</svg>
				<span class="font-semibold text-gray-700">{classroom.students.length} Students</span>
			</div>
		</div>
	</div>
	<div class="bg-gray-50 px-6 py-4">
		<button
			class="w-full rounded-md bg-indigo-600 px-4 py-2 text-white transition-colors duration-300 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2"
		>
			View Classroom
		</button>
	</div>
</div>
