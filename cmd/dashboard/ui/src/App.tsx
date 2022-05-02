import React, {useEffect, useState} from 'react';
import {IStackStyles, IStackTokens, Stack} from '@fluentui/react';
import './App.css';
import Card from "./components/Card";
import axios from "axios";
import _ from "lodash";


const stackTokens: IStackTokens = {childrenGap: 5};
const stackStyles: Partial<IStackStyles> = {
    root: {
        width: '960px',
        margin: '0 auto',
        color: '#605e5c',
    },
};

const useKeyPress = function (targetKey: string) {
    const [keyPressed, setKeyPressed] = useState(false);

    const downHandler = (ev: KeyboardEvent): any => {
        if (ev.key === targetKey) {
            setKeyPressed(true);
        }
    }

    const upHandler = (ev: KeyboardEvent): any => {
        if (ev.key === targetKey) {
            setKeyPressed(false);
        }
    };

    React.useEffect(() => {
        window.addEventListener("keydown", downHandler);
        window.addEventListener("keyup", upHandler);

        return () => {
            window.removeEventListener("keydown", downHandler);
            window.removeEventListener("keyup", upHandler);
        };
    });

    return keyPressed;
};

export const App: React.FunctionComponent = () => {
    const [selected, setSelected] = useState(undefined);
    const rightPress = useKeyPress("ArrowRight");
    const leftPress = useKeyPress("ArrowLeft");

    const [data, updateData] = useState([]);
    const [isError, setError] = useState(false);
    const [isLoading, setLoading] = useState(false);

    const [currentPage, setCurrentPage] = useState(1);

    useEffect(() => {
        if (rightPress) {
            setCurrentPage(p => p + 1)
        }
    }, [rightPress]);

    useEffect(() => {
        if (leftPress && currentPage > 1) {
            setCurrentPage(p => p - 1)
        }
    }, [leftPress]);

    useEffect(() => {
        const limit = 10
        async function fetch() {
            setError(false);
            setLoading(true);
            try {
                const {data} = await axios.get(`/list?limit=${limit}&offset=${(currentPage-1)*limit}`);
                updateData(data.data);
            } catch (e) {
                setError(true);
            }
            setLoading(false);
        }

        fetch()
    }, [currentPage])

    return (
        <Stack horizontalAlign="center" verticalAlign="center" styles={stackStyles} tokens={stackTokens}>
            {data.filter(item => {
                if ('retweeted_status' in item) {
                    item = item['retweeted_status']
                }
                return item['visible']['list_id'] == 0
            }).map(item => {
                if ('retweeted_status' in item) {
                    item = item['retweeted_status']
                }
                let images: string[] = []
                if ('pic_ids' in item) {
                    images = (item['pic_ids'] as string[]).map((id): string => {
                        if (id in item['pic_infos']) {
                            return item['pic_infos'][id]['original']['url']
                        }
                        return ''
                    }).filter(i => i != '')
                }

                return <Card
                    key={_.get(item, 'idstr', '')}
                    author={_.get(item, 'user.screen_name', '')}
                    avatar={_.get(item, 'user.profile_image_url', '')}
                    date={_.get(item, 'created_at', '')}
                    content={_.get(item, 'text_raw', '')}
                    images={images}
                />
            })}
        </Stack>
    );
};
