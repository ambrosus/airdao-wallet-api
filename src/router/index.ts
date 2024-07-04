import { Application } from "express";
import { DependencyContainer } from "tsyringe";

import healthcheck from "./healthcheck";
import watcherRouter from "./watcher.router";

export const setupRoutes = (app: Application, container: DependencyContainer) => {
    watcherRouter(app, container);
    healthcheck(app);
};