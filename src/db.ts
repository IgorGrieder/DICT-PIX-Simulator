import mongoose from "mongoose";

const MONGODB_URI = process.env.MONGODB_URI || "mongodb://localhost:27017/dict";

export async function connectDB(): Promise<typeof mongoose> {
	try {
		const conn = await mongoose.connect(MONGODB_URI);
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
