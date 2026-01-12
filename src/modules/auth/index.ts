import { jwt } from "@elysiajs/jwt";
import { Elysia } from "elysia";
import { env } from "../../config/env";
import { UserModel } from "../../models/user";
import { AuthModel } from "./model";

export const authModule = new Elysia({ prefix: "/auth" })
	.use(
		jwt({
			name: "jwt",
			secret: env.JWT_SECRET,
			exp: "7d",
		}),
	)

	// Register
	.post(
		"/register",
		async ({ jwt, body, set }) => {
			const { email, password, name } = body;

			const existingUser = await UserModel.findOne({ email });

			if (existingUser) {
				set.status = 409;
				return {
					error: "USER_ALREADY_EXISTS",
					message: "User with this email already exists",
				};
			}

			const user = await UserModel.create({
				email,
				password,
				name,
			});

			const token = await jwt.sign({
				user_id: user._id.toString(),
				email: user.email,
				name: user.name,
			});

			set.status = 201;
			return {
				token,
				user: {
					id: user._id.toString(),
					email: user.email,
					name: user.name,
				},
			};
		},
		{
			body: AuthModel.registerBody,
			response: {
				201: AuthModel.authResponse,
				409: AuthModel.error,
			},
		},
	)

	// Login
	.post(
		"/login",
		async ({ jwt, body, set }) => {
			const { email, password } = body;

			const user = await UserModel.findOne({ email });

			if (!user) {
				set.status = 401;
				return {
					error: "INVALID_CREDENTIALS",
					message: "Invalid email or password",
				};
			}

			if (user.password !== password) {
				set.status = 401;
				return {
					error: "INVALID_CREDENTIALS",
					message: "Invalid email or password",
				};
			}

			const token = await jwt.sign({
				user_id: user._id.toString(),
				email: user.email,
				name: user.name,
			});

			return {
				token,
				user: {
					id: user._id.toString(),
					email: user.email,
					name: user.name,
				},
			};
		},
		{
			body: AuthModel.loginBody,
			response: {
				200: AuthModel.authResponse,
				401: AuthModel.error,
			},
		},
	);
