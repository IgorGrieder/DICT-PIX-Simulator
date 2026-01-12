import { z } from "zod/v4";

export namespace AuthModel {
	// Request schemas
	export const registerBody = z.object({
		email: z.string().email(),
		password: z.string().min(6),
		name: z.string().min(1),
	});
	export type registerBody = z.infer<typeof registerBody>;

	export const loginBody = z.object({
		email: z.string().email(),
		password: z.string(),
	});
	export type loginBody = z.infer<typeof loginBody>;

	// Response schemas
	export const user = z.object({
		id: z.string(),
		email: z.string(),
		name: z.string(),
	});
	export type user = z.infer<typeof user>;

	export const authResponse = z.object({
		token: z.string(),
		user,
	});
	export type authResponse = z.infer<typeof authResponse>;

	export const error = z.object({
		error: z.string(),
		message: z.string(),
	});
	export type error = z.infer<typeof error>;
}
