import { singleton } from "tsyringe";
import { Request, Response } from "express";
import { WatcherService } from "./watcher.service";


@singleton()
export class WatcherNetwork {
  private readonly service: WatcherService;

  constructor(watcherService: WatcherService) {
    this.service = watcherService;
  }

  async getWatcher(req: Request, res: Response): Promise<void> {
    try {
      const { pushToken } = req.params;
      const watcher = await this.service.getWatcher(pushToken);
      res.json(watcher);
    } catch (error) {
      this.handleErrorResponse(res, error);
    }
  }

  async getWatcherHistoricalPrices(req: Request, res: Response): Promise<void> {
    try {
      const prices = await this.service.getWatcherHistoryPrices();
      res.json({ prices: JSON.parse([prices]) });
    } catch (error) {
      this.handleErrorResponse(res, error);
    }
  }

  async createWatcher(req: Request, res: Response): Promise<void> {
    try {
      const { push_token, device_id } = req.body;
      await this.service.createWatcher(push_token, device_id);
      res.json({ message: "Watcher created" });
    } catch (error) {
      this.handleErrorResponse(res, error);
    }
  }

  async updateWatcher(req: Request, res: Response): Promise<void> {
    try {
      const { threshold, tx_notification, price_notification, addresses, push_token } = req.body;
      await this.service.updateWatcher(push_token, {
        addresses,
        threshold,
        txNotification: tx_notification,
        priceNotification: price_notification,
      });
      res.json({ message: "Watcher updated" });
    } catch (error) {
      this.handleErrorResponse(res, error);
    }
  }

  async deleteWatcher(req: Request, res: Response): Promise<void> {
    try {
      const { push_token } = req.params;
      await this.service.deleteWatcher(push_token);
      res.json({ message: "Watcher deleted" });
    } catch (error) {
      this.handleErrorResponse(res, error);
    }
  }

  async deleteWatcherAddresses(req: Request, res: Response): Promise<void> {
    try {
      const { push_token, addresses } = req.body;
      await this.service.deleteWatcherAddresses(push_token, addresses);
      res.json({ message: "Watcher addresses deleted" });
    } catch (error) {
      this.handleErrorResponse(res, error);
    }
  }

  async updateWatcherPushToken(req: Request, res: Response): Promise<void> {
    try {
      const { old_push_token, new_push_token } = req.body;
      await this.service.updateWatcherPushToken(old_push_token, new_push_token);
      res.json({ message: "Watcher push token updated" });
    } catch (error) {
      this.handleErrorResponse(res, error);
    }
  }

  async watcherCallback(req: Request, res: Response): Promise<void> {
    try {
      const { id, items } = req.body;
      await this.service.watcherCallback(id, items);
    } catch (error) {
      this.handleErrorResponse(res, error);
    }
  }


  private handleErrorResponse(res: Response, error: unknown): void {
    const errorMessage =
      error instanceof Error ? error.message : "Internal server error";
    const stackTrace = error instanceof Error ? error.stack : undefined;
    res.status(400).json({ message: errorMessage, trace: stackTrace });
  }
}