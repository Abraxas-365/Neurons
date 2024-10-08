import type { RequestHandler } from '@sveltejs/kit';
import { pool, lucia } from '$lib/server/auth';
import { currentUser } from '$lib/stores/userStore';

export const POST: RequestHandler = async ({ request, cookies }) => {
	const formData = await request.formData();
	const name = formData.get('name') as string;
	const email = formData.get('email') as string;
	const role = formData.get('role') as string;
	const userId = formData.get('userId') as string;
	const googleId = formData.get('googleId') as string;
	const picture = formData.get('picture') as string;

	console.log('Received form data:', { name, email, role, userId, googleId, picture });

	if (!name || !email || !role || !userId || !googleId) {
		return new Response(JSON.stringify({ message: 'All required fields must be filled' }), {
			status: 400,
			headers: { 'Content-Type': 'application/json' }
		});
	}

	const client = await pool.connect();
	try {
		await client.query('BEGIN');

		console.log('Inserting into auth_user');
		await client.query('INSERT INTO auth_user (id, email, google_id) VALUES ($1, $2, $3)', [
			userId,
			email,
			googleId
		]);

		console.log('Inserting into users');
		await client.query(
			'INSERT INTO users (auth_user_id, name, email, role, picture) VALUES ($1, $2, $3, $4, $5)',
			[userId, name, email, role, picture]
		);

		await client.query('COMMIT');

		const session = await lucia.createSession(userId, {});
		const sessionCookie = lucia.createSessionCookie(session.id);

		cookies.set(sessionCookie.name, sessionCookie.value, {
			path: '/',
			...sessionCookie.attributes
		});

		// Set the user in the store after successful creation
		currentUser.setUser({
			id: userId,
			name,
			email,
			role,
			picture
		});

		return new Response(JSON.stringify({ success: true }), {
			status: 200,
			headers: { 'Content-Type': 'application/json' }
		});
	} catch (error) {
		await client.query('ROLLBACK');
		console.error('Detailed error:', error);
		return new Response(
			JSON.stringify({
				message: 'Failed to create user',
				details: error instanceof Error ? error.message : 'Unknown error',
				stack: error instanceof Error ? error.stack : undefined
			}),
			{
				status: 500,
				headers: { 'Content-Type': 'application/json' }
			}
		);
	} finally {
		client.release();
	}
};
