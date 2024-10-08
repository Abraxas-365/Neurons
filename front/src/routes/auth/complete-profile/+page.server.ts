import type { Actions } from './$types';
import { fail, redirect } from '@sveltejs/kit';

export const actions: Actions = {
	default: async ({ fetch, request }) => {
		const formData = await request.formData();

		const response = await fetch('/auth/complete-profile', {
			method: 'POST',
			body: formData
		});

		const result = await response.json();

		if (response.ok) {
			throw redirect(303, '/');
		} else {
			return fail(response.status, { message: result.message });
		}
	}
};
