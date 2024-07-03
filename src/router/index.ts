import { Application } from "express";

import watcherRouter from "./watcher.router";
export const setupRoutes = (app: Application) => {
    watcherRouter(app);
};