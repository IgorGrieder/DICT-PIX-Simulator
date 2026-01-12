import mongoose from "mongoose";
import { env } from "./config/env";

export async function connectDB(): Promise<typeof mongoose> {
	try {
		const conn = await mongoose.connect(env.MONGODB_URI);
		console.log(`MongoDB connected: ${conn.connection.host}`);
		return conn;
	} catch (error) {
		console.error("MongoDB connection error:", error);
		process.exit(1);
	}
}

export async function disconnectDB(): Promise<void> {
	await mongoose.disconnect();
}
