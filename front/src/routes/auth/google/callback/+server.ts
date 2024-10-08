import { OAuth2RequestError } from 'arctic';
import { generateId } from 'lucia';
import { pool, google, lucia } from '$lib/server/auth';
import type { RequestEvent } from '@sveltejs/kit';

export async function GET(event: RequestEvent): Promise<Response> {
	const code = event.url.searchParams.get('code');
	const state = event.url.searchParams.get('state');
	const storedState = event.cookies.get('google_oauth_state') ?? null;
	const codeVerifier = event.cookies.get('code_verifier') ?? null;

	if (!code || !state || !storedState || !codeVerifier || state !== storedState) {
		return new Response(null, {
			status: 400
		});
	}

	try {
		const tokens = await google.validateAuthorizationCode(code, codeVerifier);
		const googleUserResponse = await fetch('https://www.googleapis.com/oauth2/v1/userinfo', {
			headers: {
				Authorization: `Bearer ${tokens.accessToken}`
			}
		});

		if (!googleUserResponse.ok) {
			throw new Error('Failed to fetch user information from Google');
		}

		const googleUser: GoogleUser = await googleUserResponse.json();

		const client = await pool.connect();
		try {
			await client.query('BEGIN');

			const result = await client.query('SELECT * FROM auth_user WHERE google_id = $1', [
				googleUser.id
			]);
			const existingUser = result.rows[0];

			if (existingUser) {
				// User exists, create session and redirect to home
				const session = await lucia.createSession(existingUser.id, {});
				const sessionCookie = lucia.createSessionCookie(session.id);

				event.cookies.set(sessionCookie.name, sessionCookie.value, {
					path: '.',
					...sessionCookie.attributes
				});

				return new Response(null, {
					status: 302,
					headers: {
						Location: '/'
					}
				});
			} else {
				// New user, redirect to complete profile page
				const userId = generateId(15);

				return new Response(null, {
					status: 302,
					headers: {
						Location: `/auth/complete-profile?userId=${userId}&email=${encodeURIComponent(googleUser.email)}&googleId=${googleUser.id}&name=${encodeURIComponent(googleUser.name)}&picture=${encodeURIComponent(googleUser.picture)}`
					}
				});
			}
		} finally {
			client.release();
		}
	} catch (e) {
		if (e instanceof OAuth2RequestError) {
			// invalid code
			return new Response(null, {
				status: 400
			});
		}
		console.error(e);
		return new Response(null, {
			status: 500
		});
	}
}

interface GoogleUser {
	id: string;
	email: string;
	name: string;
	picture: string;
}
