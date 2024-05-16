

export interface IWatcherRepository {
    getWatcher(pushToken: string): Promise<any>; // todo define return type accordingly
    getWatcherHistoryPrices(): Promise<any>; // todo define return type accordingly
    createWatcher(pushToken: string, deviceId: string): Promise<void>;
    updateWatcher(): Promise<void>;
    deleteWatcher(): Promise<void>;
    deleteWatcherAddresses(): Promise<void>;
    watcherCallback(): Promise<void>;
    updateWatcherPushToken(): Promise<void>;
}

export class WatcherRepository implements IWatcherRepository{
    constructor() {}
    async getWatcher() {
    }
    async getWatcherHistoryPrices() {
    }
    async createWatcher() {
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
