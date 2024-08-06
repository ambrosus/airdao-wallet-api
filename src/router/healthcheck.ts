import { Application } from "express";

const healthcheck = (app: Application) => {
  app.get("/api/v1/health", (req, res) => {
    res.json({ status: "OK" });
  });
};

export default healthcheck;