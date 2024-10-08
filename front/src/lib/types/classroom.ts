// File: $lib/types/classroom.ts

import type { User } from './user';

export type Classroom = {
	id: number;
	name: string;
	teacherID: number;
	availableNeurons: number;
	createdAt: Date;
};

export type Student = {
	user: User;
	neurons: number;
};

export type ClassroomWithData = Classroom & {
	teacher: User;
	students: Student[];
};

export type UserClassroom = {
	userID: number;
	classroomID: number;
	neurons: number;
};

export type NeuronTransaction = {
	id: number;
	classroomID: number;
	userID: number;
	amount: number;
	transactionType: 'assignment' | 'return';
	createdAt: Date;
};
