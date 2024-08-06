import { singleton } from "tsyringe";
import { HistoricalNotificationModel } from "./historical-notifications.model";


@singleton()
export class HistoricalNotificationsRepository {
  constructor() {
  }

  async getHistoricalNotifications(watcherId: string) {
    return HistoricalNotificationModel.find({ watcherId }).select("-watcherId");
  }

  async addHistoricalNotification(watcherId: string, notification: Record<string, unknown>) {
    return HistoricalNotificationModel.create({ watcherId, ...notification });
  }
}