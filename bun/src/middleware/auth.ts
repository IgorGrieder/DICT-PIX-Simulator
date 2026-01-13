import { jwt } from "@elysiajs/jwt";
import { Elysia } from "elysia";
import { env } from "../config/env";

type JWTPayload = {
	user_id: string;
	email: string;
	name: string;
};

export const authMiddleware = new Elysia({ name: "auth" })
	.use(
		jwt({
			name: "jwt",
			secret: env.JWT_SECRET,
			exp: "7d",
		}),
	)
	.derive({ as: "scoped" }, async ({ jwt, headers, set }) => {
		const authorization = headers.authorization;

		if (!authorization) {
			set.status = 401;
			throw {
				error: "UNAUTHORIZED",
				message: "Authorization header is required",
			};
		}

		const payload = (await jwt.verify(authorization)) as JWTPayload | null;

		if (!payload) {
			set.status = 401;
			throw {
				error: "UNAUTHORIZED",
				message: "Invalid or expired token",
			};
		}

		return {
			user_id: payload.user_id,
		};
	});
