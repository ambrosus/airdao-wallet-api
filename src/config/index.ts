import dotenv from "dotenv";
import {env} from "process";

dotenv.config();

export const dbUrl = process.env.DATABASE_URL as string;
export const redisUrl = process.env.REDIS_URL as string;
export const tokenPriceUrl = env.TOKEN_PRICE_URL as string;
export const cgTokenPriceUrl = env.CG_TOKEN_PRICE_URL as string;
export const explorerToken = env.EXPLORER_TOKEN as string;
export const explorerUrl = env.EXPLORER_URL as string;
export const callbackUrl = env.CALLBACK_URL as string;