import {Watcher, WatcherModel} from "./watcher.model";


export class WatcherRepository {
    private readonly watcherModel: typeof WatcherModel;

    constructor(watcherModel: typeof WatcherModel) {
        this.watcherModel = watcherModel;
    }

    async getAllWatchers(filter?: Record<string, string>) {
        return this.watcherModel.find(filter!);
    }
    async getWatcher(pushToken: string): Promise<Watcher | null> {
        return this.watcherModel.findOne({pushToken});
    }

    async getWatcherByDeviceId(deviceId: string): Promise<Watcher | null> {
        return this.watcherModel.findOne({deviceId});
    }
    async createWatcher(watcher: Watcher) {
        await this.watcherModel.create(watcher);
    }

    async updateWatcher(filter: Record<string, string>, update: Record<string, unknown>) {
        return this.watcherModel.findOneAndUpdate(filter, update);
    }
    async deleteWatcher(filer: Record<string, string>) {
        return this.watcherModel.deleteOne(filer);
    }
    async deleteWatcherAddresses() {
    }
    async watcherCallback() {
    }
}
