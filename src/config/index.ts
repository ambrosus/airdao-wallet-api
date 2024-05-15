import dotenv from "dotenv";

dotenv.config()

export const dbUrl = process.env.DATABASE_URL as string;