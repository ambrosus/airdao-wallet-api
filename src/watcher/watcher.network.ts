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
            res.json(prices);
        } catch (error) {
            this.handleErrorResponse(res, error);
        }
    }

    async createWatcher(req: Request, res: Response): Promise<void> {
        try {
            const { pushToken, deviceId } = req.body;
            await this.service.createWatcher(pushToken, deviceId);
            res.json({ message: "Watcher created" });
        } catch (error) {
            this.handleErrorResponse(res, error);
        }
    }

    async updateWatcher(req: Request, res: Response): Promise<void> {
        try {
            const { pushToken } = req.params;
            const { threshold, txNotification, priceNotification, addresses } = req.body;
            await this.service.updateWatcher(pushToken, {
                addresses,
                threshold,
                txNotification,
                priceNotification,
            });
            res.json({ message: "Watcher updated" });
        } catch (error) {
            this.handleErrorResponse(res, error);
        }
    }

    async deleteWatcher(req: Request, res: Response): Promise<void> {
        try {
            const { pushToken } = req.params;
            await this.service.deleteWatcher(pushToken);
            res.json({ message: "Watcher deleted" });
        } catch (error) {
            this.handleErrorResponse(res, error);
        }
    }

    async deleteWatcherAddresses(req: Request, res: Response): Promise<void> {
        try {
            const { pushToken, addresses } = req.body;
            await this.service.deleteWatcherAddresses(pushToken, addresses);
            res.json({ message: "Watcher addresses deleted" });
        } catch (error) {
            this.handleErrorResponse(res, error);
        }
    }

    async updateWatcherPushToken(req: Request, res: Response): Promise<void> {
        try {
            const { oldPushToken, newPushToken } = req.body;
            await this.service.updateWatcherPushToken(oldPushToken, newPushToken);
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