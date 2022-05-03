import {DocumentCard, DocumentCardActivity} from "@fluentui/react/lib/DocumentCard";
import {IDocumentCardStyles, List} from "@fluentui/react";
import React from "react";
import {getTheme, ITheme, mergeStyleSets} from '@fluentui/react/lib/Styling';

const theme: ITheme = getTheme();
const {palette, fonts} = theme;

export interface ICardProps {
  author: string
  avatar: string
  date: string
  content: string
  images: string[]
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


const onRenderCell = (item: string | undefined, index: number | undefined) => {
  return (
    <div
      className={classNames.listGridTile}
      data-is-focusable
      style={{
        width: '33.33%'
      }}
    >
      <div className={classNames.listGridSizer}>
        <div className={classNames.listGridPadder}>
          <a href={item} target="_blank" rel="noreferrer"><img src={item} className={classNames.listGridImage} alt=""/></a>
        </div>
      </div>
    </div>
  );
}


export default function Card({author, avatar, date, images, content}: ICardProps) {
  const DocumentCardActivityPeople = [{
    name: author,
    profileImageSrc: avatar
  }];

  return <DocumentCard styles={cardStyles}>
    <DocumentCardActivity activity={date} people={DocumentCardActivityPeople}/>
    <pre style={{whiteSpace: 'pre-wrap', padding: '5px'}}>{content}</pre>
    <List
      style={{padding: 10}}
      items={images}
      onRenderCell={onRenderCell}
    />
  </DocumentCard>
}