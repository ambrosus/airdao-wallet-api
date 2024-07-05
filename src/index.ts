import "reflect-metadata";
import cors from "cors";
import express from "express";
import mongoose from "mongoose";

import { setupRoutes } from "./router";
import { WatcherService } from "./watcher";
import { createContainer } from "./common";
import { ExplorerService } from "./explorer";
import { appPort, dbUrl, ONE_DAY_IN_MS } from "./config";
import { CgPriceWatcher, ApiPriceWatcher, PriceWatcher } from "./price-watchers";

async function main() {
    if (!dbUrl) {
        throw new Error("DB URL not found");
    }

    const container = await createContainer();

    await mongoose.connect(dbUrl);

    const app = express();
    app.use(cors());
    app.use(express.json());
    app.use(express.urlencoded({ extended: true }));

    setupRoutes(app, container);

    const watcherService = container.resolve(WatcherService);
    const explorerService = container.resolve(ExplorerService);
    await explorerService.initService();

    const cgPriceWatcher = container.resolve(CgPriceWatcher);
    const apiPriceWatcher = container.resolve(ApiPriceWatcher);
    const priceWatcher = container.resolve(PriceWatcher);

    // Run deleteWatchersWithStaleData every 24 hours for check and delete stale data
    setInterval(
        async () => await watcherService.deleteWatchersWithStaleData(),
        ONE_DAY_IN_MS
    );

    await Promise.all([
        cgPriceWatcher.run(),
        apiPriceWatcher.run(),
        watcherService.subscribeToExplorer(),
        explorerService.checkService(),
        priceWatcher.run()
    ]);

    app.listen(appPort, () => {
        console.log(`Server is running on port ${appPort}`);
    });
}

main().then((res) => console.log(res));
