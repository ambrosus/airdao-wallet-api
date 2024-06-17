import { Request, Response } from "express";
import { IWatcherService } from "../watcher/watcher.service";

export interface IHandlers {
    getWatcher(req: Request, res:Response): Promise<void>;
    getWatcherHistoryPrices(req: Request, res:Response): Promise<void>;
    createWatcher(req: Request, res:Response): Promise<void>;
    updateWatcher(req: Request, res:Response): Promise<void>;
    deleteWatcher(req: Request, res:Response): Promise<void>;
    deleteWatcherAddresses(req: Request, res:Response): Promise<void>;
    watcherCallback(req: Request, res:Response): Promise<void>;
    updateWatcherPushToken(req: Request, res:Response): Promise<void>;
}

export class WatcherHandlers implements IHandlers {
    constructor(private readonly watcherService: IWatcherService) {}

    async getWatcher(req: Request, res:Response ) {
        try {
            const {pushToken} = req.params;
            if (!pushToken) {
                res.status(400).json({message: "Token is required"});
                return;
            }

            const watcher = await this.watcherService.getWatcher(pushToken);

            res.json(watcher);
        } catch (e) {
            this.handleError(res, e as Error);
        }
    }
    async getWatcherHistoryPrices(req: Request, res:Response ) {
        try {

        } catch (e) {
            this.handleError(res, e as Error);
        }
    }
    async createWatcher(req: Request, res:Response ) {
        try {
            const {pushToken, deviceId} = req.body;
            if (!pushToken || !deviceId) {
                // todo check if deviceId should be required
                res.status(400).json({message: "PushToken and deviceId are required"});
            }
            const watcher = await this.watcherService.createWatcher(pushToken, deviceId);
            res.json(watcher);
        } catch (e) {
            this.handleError(res, e as Error);
        }
    }
    async updateWatcher(req: Request, res:Response ) {
        try {

        } catch (e) {
            this.handleError(res, e as Error);
        }
    }
    async deleteWatcher(req: Request, res:Response ) {
        try {

        } catch (e) {
            this.handleError(res, e as Error);
        }
    }
    async deleteWatcherAddresses(req: Request, res:Response ) {
        try {

        } catch (e) {
            this.handleError(res, e as Error);
        }
    }
    async watcherCallback(req: Request, res:Response ) {
        try {

        } catch (e) {
            this.handleError(res, e as Error);
        }
    }
    async updateWatcherPushToken(req: Request, res:Response ) {
        try {

        } catch (e) {
            this.handleError(res, e as Error);
        }
    }

    private handleError(res: Response, error: Error) {
        console.error(error);
        res.status(501).json({ message: error.message, trace: error.stack });
    }
}
