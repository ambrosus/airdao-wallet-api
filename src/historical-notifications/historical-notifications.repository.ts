import { singleton } from "tsyringe";
import { HistoricalNotificationModel } from "./historical-notifications.model";


@singleton()
export class HistoricalNotificationsRepository {
    constructor(private readonly model: typeof HistoricalNotificationModel) {
    }

    async getHistoricalNotifications(watcherId: string) {
        return this.model.find({ watcherId }).select("-watcherId");
    }
}