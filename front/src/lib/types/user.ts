// File: $lib/types/user.ts
export type User = {
	id: number;
	name: string;
	email: string;
	role: 'teacher' | 'student';
	createdAt: Date;
};
