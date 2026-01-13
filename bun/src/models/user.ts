import mongoose, { type Document, Schema } from "mongoose";
import { AuthModel } from "../modules/auth/model";

export type UserDocument = AuthModel.registerBody &
	Document & {
		createdAt: Date;
		updatedAt: Date;
	};

const userSchema = new Schema<UserDocument>(
	{
		email: {
			type: String,
			required: true,
			unique: true,
			lowercase: true,
			trim: true,
		},
		password: {
			type: String,
			required: true,
		},
		name: {
			type: String,
			required: true,
		},
	},
	{
		timestamps: true,
	},
);

export const UserModel = mongoose.model<UserDocument>("User", userSchema);
