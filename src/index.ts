import cors from "cors";
import Redis from "ioredis";
import express from "express";
import mongoose from "mongoose";
import { container } from "tsyringe";
import { setupRoutes } from "./router";
import { dbUrl, redisUrl } from "./config";
import { WatcherService } from "./watcher";
import { CgPriceWatcher, ApiPriceWatcher } from "./price-watchers";
import { ExplorerService } from "./explorer";


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

    setupRoutes(app);

    const watcherService = container.resolve(WatcherService);
    const explorerService = container.resolve(ExplorerService);

    const cgPriceWatcher = container.resolve(CgPriceWatcher);
    const apiPriceWatcher = container.resolve(ApiPriceWatcher);

    await Promise.all([
        cgPriceWatcher.run(),
        apiPriceWatcher.run(),
        watcherService.subscribeToExplorer(),
        explorerService.checkService()
    ]);
}

main().then((res) => console.log(res));
