import React from "react";
import {Card, Avatar, Divider, Image} from 'antd';
import moment from 'moment';

export interface IImageProps {
  thumbnail: string
  origin: string
}

export interface IUserProps {
  profile_image_url: string
  screen_name: string
  id: number
}

export interface IArchiveImg {
  thumb: string
  origin: string
}

export interface ITweetProps {
  text: string
  text_raw: string
  mid: string
  mblogid: string
  created_at: string
  user: IUserProps
  visible: { listId: number }
  pic_ids: string[]
  archiveVideo?: string
  archiveImages: { [key: string]: IArchiveImg }
  retweeted_status: ITweetProps
}

const renderTweetHead = (tweet: ITweetProps) => {
  if (!tweet.user) {
    return null
  }
  const tweetHref = `https://weibo.com/${tweet.user.id}/${tweet.mblogid}`
  const userHref = `https://weibo.com/u/${tweet.user.id}`
  return <div style={{padding: 10}}>
    <Card.Meta
      avatar={<a href={userHref} target='_blank' rel='noreferrer'><Avatar src={tweet.user.profile_image_url}/></a>}
      title={tweet.user.screen_name}
      description={<div>
        <span>{moment(tweet.created_at).format('YYYY-MM-DD hh:mm:ss')} -  </span>
        <a href={tweetHref} target='_blank' rel='noreferrer'>{tweet.mid}</a>
      </div>}
    />
    <Divider style={{margin: '10px 0'}}/>
    <pre style={{whiteSpace: 'pre-wrap', padding: '5px', fontFamily: 'inherit'}}><div dangerouslySetInnerHTML={{__html: tweet.text}} /></pre>
  </div>
}

const getImages = (tweet: ITweetProps): IImageProps[] => {
  if (tweet.retweeted_status) {
    tweet = tweet.retweeted_status
  }
  if (!tweet.pic_ids) {
    return []
  }
  return tweet.pic_ids?.map((id): IImageProps | undefined => {
    if (id in tweet.archiveImages) {
      return {
        thumbnail: `/api/image/${id}-thumb.jpg`,
        origin: `/api/image/${id}.jpg`,
      }
    }
    return undefined
  }).filter((i): i is IImageProps => !!i)
}

const renderImages = (tweet: ITweetProps) => {
  const images = getImages(tweet)
  let width: number | string = '33.3%'
  if (images.length === 1) {
    width = '100%'
  } else if (images.length === 2) {
    width = '50%'
  }
  return <Image.PreviewGroup>
    {images.map(img => <Image
	  key={img.origin}
      width={width}
      src={images.length === 1 ? img.origin : img.thumbnail}
      preview={{src: img.origin}}
    />)}
  </Image.PreviewGroup>
}

export default function Tweet({data}: { data: ITweetProps }) {
  const originTweet = data.retweeted_status ? data.retweeted_status : data
  return <div style={{marginTop: 15}}>
    <Card
      style={{width: 800}}
      bodyStyle={{padding: 10}}
    >
      {renderTweetHead(data)}
      {data.retweeted_status ? <div style={{background: "#F0F1F4"}}>
        {renderTweetHead(data.retweeted_status)}
        {renderImages(data.retweeted_status)}
      </div> : renderImages(data)}
      {originTweet.archiveVideo ?
        <video controls width="100%">
          <source src={`/api/video/${originTweet.archiveVideo}`} type="video/mp4"/>
        </video> : null}
    </Card>
  </div>
}
