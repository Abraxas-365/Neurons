// $lib/stores/userStore.ts
import { writable } from 'svelte/store';
import type { User } from '$lib/types/user';
import { api } from '$lib/api';
import { browser } from '$app/environment';

const createUserStore = () => {
	const { subscribe, set, update } = writable<User | null>(null);

	return {
		subscribe,
		setUser: (user: User) => {
			set(user);
			if (browser) {
				localStorage.setItem('currentUser', JSON.stringify(user));
			}
		},
		clearUser: () => {
			set(null);
			if (browser) {
				localStorage.removeItem('currentUser');
			}
		},
		login: async () => {
			if (!browser) return;
			try {
				const user = await api.get('/users/me');
				set(user);
				localStorage.setItem('currentUser', JSON.stringify(user));
			} catch (error) {
				console.error('Error fetching user data:', error);
				set(null);
			}
		},
		logout: async () => {
			if (!browser) return;
			try {
				await api.post('/auth/logout', {});
				set(null);
				localStorage.removeItem('currentUser');
			} catch (error) {
				console.error('Error logging out:', error);
			}
		},
		initializeFromStorage: () => {
			if (browser) {
				const storedUser = localStorage.getItem('currentUser');
				if (storedUser) {
					set(JSON.parse(storedUser));
				}
			}
		}
	};
};

export const currentUser = createUserStore();

if (browser) {
	currentUser.initializeFromStorage();
}
