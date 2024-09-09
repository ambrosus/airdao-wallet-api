import dotenv from "dotenv";
import { env } from "process";

dotenv.config();

type AppEnv = "dev" | "prod";

export const dbUrl = process.env.MONGO_DB_URL as string;
export const redisUrl = process.env.REDIS_URL as string;
export const firebaseCredPath = process.env.FIREBASE_CRED_PATH as string;
export const tokenPriceUrl = env.TOKEN_PRICE_URL as string;
export const cgTokenPriceUrl = env.CG_TOKEN_PRICE_URL as string;
export const explorerToken = env.EXPLORER_TOKEN as string;
export const explorerUrl = env.EXPLORER_API as string;
export const callbackUrl = env.CALLBACK_URL as string;
export const appPort = Number(env.PORT) || 5001;
export const appEnv = env.APP_ENV as AppEnv || "dev";
export const rpcUrl = env.RPC_URL as string;
export const rewardsBankAddress = env.REWARDS_BANK_ADDRESS as string;

export const androidChannel = env.ANDROID_CHANNEL_NAME as string;

export const ONE_DAY_IN_MS = 24 * 60 * 60 * 1000;

export const notificationsTitleConfig = {
  dev: {
    priceAlert: "Price Alert Dev",
    txAlert: "AMB-DevNet Tx Alert"
  },
  test: {
    priceAlert: "Price Alert Test",
    txAlert: "AMB-TestNet Tx Alert"
  },
  prod: {
    priceAlert: "Price Alert",
    txAlert: "AMB-Net Tx Alert"
  }
};