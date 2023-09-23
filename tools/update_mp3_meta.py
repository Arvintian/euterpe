from mutagen.id3 import ID3, TIT2, TALB, TPE1
import argparse
import os

def traverse_directory(path,tpe1):
    dir_name = os.path.basename(path)
    for file_name in os.listdir(path):
        full_path = os.path.join(path, file_name)
        if os.path.isdir(full_path):
            traverse_directory(full_path,tpe1)
        else:
            tit2,talb = file_name,dir_name
            if file_name.endswith(".mp3"):
                tit2 = file_name.replace(".mp3","")
            tags = ID3(full_path)
            # 标题
            tags["TIT2"] = TIT2(encoding=3, text=tit2)
            # 专辑
            tags["TALB"] = TALB(encoding=3, text=talb)
            # 艺术家
            tags["TPE1"] = TPE1(encoding=3, text=tpe1)
            tags.save(v2_version=3)
            print(tpe1,tit2,talb,full_path)
            # exit()

def main():
    parser = argparse.ArgumentParser("MP3Meta")
    parser.add_argument("path")
    parser.add_argument("tpe1")
    args = parser.parse_args()
    traverse_directory(args.path,args.tpe1)


if __name__=="__main__":
    main()
