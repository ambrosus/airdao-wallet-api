import express from "express";
import mongoose from "mongoose";
import cors from "cors";
import Redis from "ioredis";
import {dbUrl, redisUrl} from "./config";
import {CgPriceWatcher, ApiPriceWatcher} from "./price-watchers";
import {container} from "tsyringe";


async function main() {
    if (!dbUrl) {
        throw new Error("DB URL not found");
    }

    const cacheStorage = new Redis(redisUrl);
    await mongoose.connect(dbUrl);

    const app = express();
    app.use(cors());
    app.use(express.json());
    app.use(express.urlencoded({extended: true}));

    const cgPriceWatcher = container.resolve(CgPriceWatcher);
    const apiPriceWatcher = container.resolve(ApiPriceWatcher);

    await Promise.all([
        cgPriceWatcher.run(),
        apiPriceWatcher.run(),
    ]);
}

main().then((res) => console.log(res));
