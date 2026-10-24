import landing from './landing'
import common from './common'
import dashboard from './dashboard'
import channelMonitorV2 from './channelMonitorV2'
import batchImage from './batchImage'
import imageGeneration from './imageGeneration'
import helpCenter from './helpCenter'
import downloads from './downloads'
import supportContact from './supportContact'
import admin from './admin'
import misc from './misc'

export default {
  ...landing,
  ...common,
  ...dashboard,
  ...channelMonitorV2,
  ...batchImage,
  ...imageGeneration,
  ...supportContact,
  ...helpCenter,
  ...downloads,
  admin,
  ...misc,
}
