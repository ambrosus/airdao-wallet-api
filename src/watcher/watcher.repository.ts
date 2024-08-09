import { singleton } from "tsyringe";
import { Watcher, WatcherModel } from "./watcher.model";

@singleton()
export class WatcherRepository {

  constructor() {
  }

  async getAllWatchers(filter?: Record<string, string | Record<string, unknown> | unknown>) {
    return WatcherModel.find(filter!);
  }

  async getWatcher(pushToken: string): Promise<Watcher | null> {
    return WatcherModel.findOne({ pushToken });
  }

  async getWatcherWithTxNotifications(id: string) {
    return WatcherModel.findOne({ _id: id, txNotification: "ON" });
  }

  async getWatcherByDeviceId(deviceId: string): Promise<Watcher | null> {
    return WatcherModel.findOne({ deviceId });
  }

  async createWatcher(watcher: Watcher) {
    await WatcherModel.create(watcher);
  }

  async updateWatcher(filter: Record<string, string>, update: Record<string, unknown>) {
    return WatcherModel.findOneAndUpdate(filter, update).exec();
  }

  async deleteWatcher(filer: Record<string, string>) {
    return WatcherModel.deleteOne(filer);
  }
}
