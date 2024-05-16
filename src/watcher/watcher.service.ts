import {IWatcherRepository} from "./watcher.repository";

export interface IWatcherService {
    getWatcher(pushToken: string): Promise<any>; // todo define return type accordingly
    getWatcherHistoryPrices(): Promise<any>; // todo define return type accordingly
    createWatcher(pushToken: string, deviceId: string): Promise<void>;
    updateWatcher(): Promise<void>;
    deleteWatcher(): Promise<void>;
    deleteWatcherAddresses(): Promise<void>;
    watcherCallback(): Promise<void>;
    updateWatcherPushToken(): Promise<void>;
}

export class WatcherService implements IWatcherService {

    // todo pass dependencies IWatcherRepository IHistoryRepository, IExplorerService, INotificationSender
    constructor(private readonly watcherRepository: IWatcherRepository) {}

    async getWatcher(pushToken: string) {

    }
    async getWatcherHistoryPrices() {

    }
    async createWatcher(pushToken: string, deviceId: string) {

    }
    async updateWatcher() {

    }
    async deleteWatcher() {

    }
    async deleteWatcherAddresses() {

    }
    async watcherCallback() {

    }
    async updateWatcherPushToken() {

    }
}
