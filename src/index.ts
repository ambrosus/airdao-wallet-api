import cors from "cors";
import Redis from "ioredis";
import mongoose from "mongoose";
import { container } from "tsyringe";
import express, { Request, Response } from "express";

import { setupRoutes } from "./router";
import { WatcherService } from "./watcher";
import { ExplorerService } from "./explorer";
import { dbUrl, ONE_DAY_IN_MS, redisUrl } from "./config";
import { CgPriceWatcher, ApiPriceWatcher } from "./price-watchers";



async function main() {
    if (!dbUrl) {
        throw new Error("DB URL not found");
    }

    const cacheStorage = new Redis(redisUrl);
    await mongoose.connect(dbUrl);

    const app = express();
    app.use(cors());
    app.use(express.json());
    app.use(express.urlencoded({ extended: true }));
    app.use((req: Request, res: Response, next) => {
        res.setHeader("X-Custom-Header", "AIRDAO-Mobile-Api");
        next();
    });

    setupRoutes(app);

    const watcherService = container.resolve(WatcherService);
    const explorerService = container.resolve(ExplorerService);

    const cgPriceWatcher = container.resolve(CgPriceWatcher);
    const apiPriceWatcher = container.resolve(ApiPriceWatcher);

    // Run deleteWatchersWithStaleData every 24 hours for check and delete stale data
    setInterval(
        async () => await watcherService.deleteWatchersWithStaleData(),
        ONE_DAY_IN_MS
    );

    await Promise.all([
        cgPriceWatcher.run(),
        apiPriceWatcher.run(),
        watcherService.subscribeToExplorer(),
        explorerService.checkService()
    ]);
}

main().then((res) => console.log(res));
