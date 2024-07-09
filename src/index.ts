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
    console.log("Container created");

    await mongoose.connect(dbUrl);
    console.log("DB connected");

    const app = express();
    app.use(cors());
    app.use(express.json());
    app.use(express.urlencoded({ extended: true }));
    console.log("App created");

    setupRoutes(app, container);
    console.log("Routes are set up");

    // const watcherService = container.resolve(WatcherService);
    // console.log("Watcher service resolved");
    // const explorerService = container.resolve(ExplorerService);
    // console.log("Explorer service resolved");
    // await explorerService.initService();
    // console.log("Explorer service initiated");

    const cgPriceWatcher = container.resolve(CgPriceWatcher);
    // const apiPriceWatcher = container.resolve(ApiPriceWatcher);
    // const priceWatcher = container.resolve(PriceWatcher);

    // Run deleteWatchersWithStaleData every 24 hours for check and delete stale data
    // setInterval(
    //     async () => await watcherService.deleteWatchersWithStaleData(),
    //     ONE_DAY_IN_MS
    // );
    console.log("Watchers with stale data deleted");

    await Promise.all([
        cgPriceWatcher.run(),
        // apiPriceWatcher.run(),
        // watcherService.subscribeToExplorer(),
        // explorerService.checkService(),
        // priceWatcher.run()
    ]);

    app.listen(appPort, () => {
        console.log(`Server is running on port ${appPort}`);
    });
}

main().then((res) => console.log(res));
