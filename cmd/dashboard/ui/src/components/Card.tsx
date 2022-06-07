import {DocumentCard, DocumentCardActivity} from "@fluentui/react/lib/DocumentCard";
import {IDocumentCardStyles, List} from "@fluentui/react";
import React from "react";
import {getTheme, ITheme, mergeStyleSets} from '@fluentui/react/lib/Styling';
import moment from 'moment';

const theme: ITheme = getTheme();
const {palette, fonts} = theme;

export interface IImageProps {
  thumbnail?: string
  origin?: string
}

export interface IUserProps {
  profile_image_url: string
  screen_name: string
}

export interface IArchiveImg {
  thumb: string
  origin: string
}

export interface ITweetProps {
  text_raw: string
  mid: string
  created_at: string
  user: IUserProps
  visible: { listId: number }
  pic_ids: string[]
  video?: string
  archiveImages: { [key: string]: IArchiveImg }
  retweeted_status: ITweetProps
}

const cardStyles: Partial<IDocumentCardStyles> = {
  root: {minWidth: '800px', userSelect: 'default'}
}

const classNames = mergeStyleSets({
  listGrid: {
    overflow: 'hidden',
    fontSize: 0,
    position: 'relative',
  },
  listGridTile: {
    textAlign: 'center',
    outline: 'none',
    position: 'relative',
    float: 'left',
    background: palette.neutralLighter,
    selectors: {
      'focus:after': {
        content: '',
        position: 'absolute',
        left: 2,
        right: 2,
        top: 2,
        bottom: 2,
        boxSizing: 'border-box',
        border: `1px solid ${palette.white}`,
      },
    },
  },
  listGridSizer: {
    paddingBottom: '100%',
  },
  listGridPadder: {
    position: 'absolute',
    left: 2,
    top: 2,
    right: 2,
    bottom: 2,
  },
  listGridLabel: {
    background: 'rgba(0, 0, 0, 0.3)',
    color: '#FFFFFF',
    position: 'absolute',
    padding: 10,
    bottom: 0,
    left: 0,
    width: '100%',
    fontSize: fonts.small.fontSize,
    boxSizing: 'border-box',
  },
  listGridImage: {
    position: 'absolute',
    top: 0,
    left: 0,
    width: '100%',
    maxHeight: '100%',
    objectFit: 'cover',
  },
});


const onRenderCell = (item: IImageProps | undefined, index: number | undefined) => {
  return (
    <div
      className={classNames.listGridTile}
      data-is-focusable={true}
      style={{
        width: '33.33%'
      }}
    >
      <div className={classNames.listGridSizer}>
        <div className={classNames.listGridPadder}>
          <a href={item?.origin} target="_blank" rel="noreferrer">
            <img
              src={item?.thumbnail}
              className={classNames.listGridImage}
              alt=""/>
          </a>
        </div>
      </div>
    </div>
  );
}

const renderTweetHead = (tweet: ITweetProps) => {
  return <div>
    <DocumentCardActivity
      activity={`${moment(tweet.created_at).format('YYYY-MM-DD hh:mm:ss')}  -  ${tweet.mid}`}
      people={[{
        name: tweet.user.screen_name,
        profileImageSrc: tweet.user.profile_image_url,
      }]}/>
    <pre style={{whiteSpace: 'pre-wrap', padding: '5px', fontFamily: 'inherit'}}>{tweet.text_raw}</pre>
  </div>
}

const getImages = (tweet: ITweetProps): IImageProps[] => {
  if (tweet.retweeted_status) {
    tweet = tweet.retweeted_status
  }
  return tweet.pic_ids.map((id): IImageProps | undefined => {
    if (id in tweet.archiveImages) {
      const t = tweet.archiveImages[id]
      return {
        thumbnail: `/api/resource?key=${t.thumb}`,
        origin: `/api/resource?key=${t.origin}`,
      }
    }
    return undefined
  }).filter((i): i is IImageProps => !!i)
}

export default function Card({data}: { data: ITweetProps }) {
  const originTweet = data.retweeted_status ? data.retweeted_status : data
  return <div style={{marginTop: '20px'}}>
    <DocumentCard styles={cardStyles}>
      {renderTweetHead(data)}
      {data.retweeted_status ? <div style={{background: "#F0F1F4"}}>
        {renderTweetHead(data.retweeted_status)}
        <List
          style={{padding: 9, overflow: 'auto'}}
          items={getImages(data)}
          onRenderCell={onRenderCell}
        />
      </div> : <List
        style={{padding: 9, overflow: 'auto'}}
        items={getImages(data)}
        onRenderCell={onRenderCell}
      />}
      {originTweet.video ?
        <video controls width="100%">
          <source src={originTweet.video} type="video/mp4"/>
        </video> : null
      }
    </DocumentCard>
  </div>
}