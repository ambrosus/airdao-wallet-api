import { Application } from "express";
import healthcheck from "./healthcheck";
import watcherRouter from "./watcher.router";

export const setupRoutes = (app: Application) => {
    watcherRouter(app);
    healthcheck(app);
};