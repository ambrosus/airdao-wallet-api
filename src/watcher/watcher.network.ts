import { Request, Response } from "express";
import { WatcherService } from "./watcher.service";

export class WatcherNetwork {

  constructor(private readonly watcherService: WatcherService) {}

  async getWatcher(req: Request, res:Response ) {
    try {
      const {pushToken} = req.params;
      if (!pushToken) return res.status(400).json({message: "Token is required"});

      const watcher = await this.watcherService.getWatcher(pushToken);

      res.json(watcher);
    } catch (e) {
      this.handleError(res, e as Error)
    }
  }
  async getWatcherHistoryPrices(req: Request, res:Response ) {
    try {
      
    } catch (e) {
      this.handleError(res, e as Error)
    }
  }
  async createWatcher(req: Request, res:Response ) {
    try {
      const {pushToken, deviceId} = req.body;
        if (!pushToken || !deviceId) return res.status(400).json({message: "PushToken and deviceId are required"});
        const watcher = await this.watcherService.createWatcher(pushToken, deviceId);
        res.json(watcher);
    } catch (e) {
      this.handleError(res, e as Error)
    }
  }
  async updateWatcher(req: Request, res:Response ) {
    try {
      
    } catch (e) {
      this.handleError(res, e as Error)
    }
  }
  async deleteWatcher(req: Request, res:Response ) {
    try {
      
    } catch (e) {
      this.handleError(res, e as Error)
    }
  }
  async deleteWatcherAddresses(req: Request, res:Response ) {
    try {
      
    } catch (e) {
      this.handleError(res, e as Error)
    }
  }
  async watcherCallback(req: Request, res:Response ) {
    try {
      
    } catch (e) {
      this.handleError(res, e as Error)
    }
  }
  async updateWatcherPushToken(req: Request, res:Response ) {
    try {
      
    } catch (e) {
      this.handleError(res, e as Error)
    }
  }

  private handleError(res: Response, error: Error) {
    console.error(error);
    res.status(501).json({ message: error.message, trace: error.stack });
  }
}