import axios from "axios";
import {singleton} from "tsyringe";
import { callbackUrl, explorerToken, explorerUrl } from "../config";

@singleton()
export class ExplorerService {
    async initService() {
        await axios.post(`${explorerUrl}/watch`, {"id": explorerToken, "action": "init", "url": callbackUrl});
    }

    async subscribeAddresses(addresses: string[]) {
        await axios.post(`${explorerUrl}/watch`, {"id": explorerToken, "addresses": addresses, "action": "subscribe"});
    }

    async unsubscribeAddresses(addresses: string[]) {
        await axios.post(`${explorerUrl}/watch`, {"id": explorerToken, "addresses": addresses, "action": "unsubscribe"});
    }

    async checkService() {
// 		tries := 6
// 		for {
// 			req.Reset()
// 			req.WriteString("{\"id\":\"")
// 			req.WriteString(s.explorerToken)
// 			req.WriteString("\",\"action\":\"check\"}")
// 			if err := s.doRequest(fmt.Sprintf("%s/watch", s.explorerUrl), &req, nil); err != nil {
// 				s.logger.Errorln(err)
// 				if tries != 0 {
// 					tries--
// 					time.Sleep(5 * time.Second)
// 					continue
// 				}
// 				break
// 			}
// 			time.Sleep(30 * time.Second)
// 			tries = 6
// 		}
    }
}
