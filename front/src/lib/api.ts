import { browser } from '$app/environment';

// Use an environment variable for the API URL
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';

async function fetchAPI(endpoint: string, options: RequestInit = {}) {
	if (!browser) {
		console.warn('API call attempted on server side');
		return null;
	}

	const url = `${API_BASE_URL}${endpoint}`;
	const response = await fetch(url, {
		...options,
		credentials: 'include',
		headers: {
			...options.headers,
			'Content-Type': 'application/json'
		}
	});

	if (!response.ok) {
		throw new Error(`API call failed: ${response.statusText}`);
	}

	return response.json();
}

export const api = {
	get: (endpoint: string) => fetchAPI(endpoint),
	post: (endpoint: string, data: any) =>
		fetchAPI(endpoint, {
			method: 'POST',
			body: JSON.stringify(data)
		})
	// Add other methods (PUT, DELETE, etc.) as needed
};
